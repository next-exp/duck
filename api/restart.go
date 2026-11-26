package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"connectrpc.com/connect"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
	control "github.com/jmbenlloch/next_duck/rpc/control"
	"github.com/jmbenlloch/next_duck/rpc/control/controlconnect"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type restartTarget struct {
	name, host, role, ip string
	port                 int32
	state, detail        string
}
type restartOperation struct {
	id, failure string
	targets     []restartTarget
}

func (op *restartOperation) snapshot() []*pb.RestartTargetStatus {
	var result []*pb.RestartTargetStatus
	for _, t := range op.targets {
		result = append(result, &pb.RestartTargetStatus{Name: t.name, Host: t.host, Role: t.role, State: t.state, Detail: t.detail})
	}
	return result
}
func (s *DuckAPIServer) beginRestart(ctx context.Context) (*connect.Response[pb.RestartServicesResponse], error) {
	rt := &s.runTransition
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.state != TransitionIdle && rt.state != TransitionError {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("run control is %s", stateString(rt.state)))
	}
	run, err := s.queries.GetLatestRunWithTimestamp(ctx)
	if err != nil {
		return nil, err
	}
	if run.Start.Valid && !run.Stop.Valid {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("acquisition is active"))
	}
	var token [16]byte
	if _, err = rand.Read(token[:]); err != nil {
		return nil, err
	}
	id := fmt.Sprintf("%x", token)
	s.restart = restartOperation{id: id}
	rt.state = TransitionRestarting
	rt.errMsg = ""
	go s.runRestart()
	return connect.NewResponse(&pb.RestartServicesResponse{Success: true, Message: "restarting services", OperationId: id}), nil
}
func (s *DuckAPIServer) runRestart() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	err := s.restartTargets(ctx)
	rt := &s.runTransition
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.state = TransitionIdle
	if err != nil {
		s.restart.failure = err.Error()
	}
}
func (s *DuckAPIServer) restartTargets(ctx context.Context) error {
	gdcs, err := s.queries.ListGDCs(ctx)
	if err != nil {
		return err
	}
	ldcs, err := s.queries.ListLDCs(ctx)
	if err != nil {
		return err
	}
	var targets []restartTarget
	for _, g := range gdcs {
		if g.Enabled.Bool {
			targets = append(targets, restartTarget{name: g.Name.String, host: g.Hostname.String, role: "gdc", ip: g.Ip.String, port: g.GrpcPort.Int32, state: "queued"})
		}
	}
	for _, l := range ldcs {
		if l.Enabled.Bool {
			targets = append(targets, restartTarget{name: l.Name.String, host: l.Hostname.String, role: "ldc", ip: l.Ip.String, port: l.GrpcPort.Int32, state: "queued"})
		}
	}
	if len(targets) == 0 {
		return errors.New("no enabled acquisition services")
	}
	s.runTransition.mu.Lock()
	s.restart.targets = append([]restartTarget(nil), targets...)
	s.runTransition.mu.Unlock()
	update := func(i int, state, detail string) {
		s.runTransition.mu.Lock()
		defer s.runTransition.mu.Unlock()
		s.restart.targets[i].state = state
		s.restart.targets[i].detail = detail
	}
	// Check the entire detector before sending the first restart command.
	for i, t := range targets {
		if err := t.ready(ctx); err != nil {
			update(i, "failed", err.Error())
			return fmt.Errorf("%s preflight: %w", t.name, err)
		}
	}
	// Use the control IP to group co-located services even when their SSH aliases differ.
	byHost := make(map[string][]int)
	for i, t := range targets {
		byHost[t.ip] = append(byHost[t.ip], i)
	}
	slots := make(chan struct{}, 3)
	failures := make([]error, len(targets))
	var wg sync.WaitGroup
	for _, indices := range byHost {
		wg.Add(1)
		go func(indices []int) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			for _, i := range indices {
				t := targets[i]
				if err := ctx.Err(); err != nil {
					update(i, "timed out", err.Error())
					failures[i] = err
					continue
				}
				update(i, "restarting", "")
				targetCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				err := t.restartSSH(targetCtx)
				if err == nil {
					for {
						err = t.ready(targetCtx)
						if err == nil || targetCtx.Err() != nil {
							break
						}
						select {
						case <-targetCtx.Done():
						case <-time.After(500 * time.Millisecond):
						}
					}
				}
				timedOut := targetCtx.Err() != nil
				cancel()
				if err != nil {
					state := "failed"
					if timedOut {
						state = "timed out"
						if probe := t.ready(ctx); probe == nil {
							err = fmt.Errorf("restart unconfirmed; service currently INITIALIZED: %w", err)
						}
					}
					update(i, state, err.Error())
					failures[i] = fmt.Errorf("%s: %w", t.name, err)
					continue
				}
				update(i, "ready", "")
			}
		}(indices)
	}
	wg.Wait()
	return errors.Join(failures...)
}
func (t restartTarget) ready(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	addr := net.JoinHostPort(t.ip, strconv.Itoa(int(t.port)))
	client := controlconnect.NewRunControlClient(http.DefaultClient, "http://"+addr)
	reply, err := client.GetState(ctx, connect.NewRequest(&control.DuckRequest{}))
	if err != nil {
		return err
	}
	if reply.Msg.Message != "INITIALIZED" {
		return fmt.Errorf("state %s", reply.Msg.Message)
	}
	return nil
}
func restartCommand(profile, role string) (string, error) {
	if role != "gdc" && role != "ldc" {
		return "", fmt.Errorf("unsupported service %q", role)
	}
	switch profile {
	case "next100":
		return "docker compose --project-directory /home/duck/duck-daq -p duck-daq restart " + role, nil
	case "duck-prod":
		return "docker container restart duck-prod-" + role + "-1", nil
	case "", "legacy":
		return "/home/dateuser/duck/restart_" + role + ".sh", nil
	default:
		return "", fmt.Errorf("unknown restart profile %q", profile)
	}
}
func (t restartTarget) restartSSH(ctx context.Context) error {
	profile := os.Getenv("DUCK_RESTART_PROFILE")
	cmd, err := restartCommand(profile, t.role)
	if err != nil {
		return err
	}
	user := os.Getenv("DUCK_RESTART_USER")
	if user == "" {
		user = "dateuser"
		if profile == "next100" {
			user = "duck"
		}
	}
	dir := os.Getenv("DUCK_RESTART_SSH_DIR")
	if dir == "" {
		dir = filepath.Join(os.Getenv("HOME"), ".ssh")
	}
	key, err := os.ReadFile(filepath.Join(dir, "id_rsa"))
	if err != nil {
		return err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return err
	}
	verify, err := knownhosts.New(filepath.Join(dir, "known_hosts"))
	if err != nil {
		return err
	}
	port := os.Getenv("DUCK_RESTART_SSH_PORT")
	if port == "" {
		port = "22"
	}
	addr := net.JoinHostPort(t.host, port)
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return err
		}
	}
	c, ch, req, err := ssh.NewClientConn(conn, addr, &ssh.ClientConfig{User: user, Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)}, HostKeyCallback: verify})
	if err != nil {
		return err
	}
	client := ssh.NewClient(c, ch, req)
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	return session.Run(cmd)
}
