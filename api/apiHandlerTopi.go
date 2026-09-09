package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"connectrpc.com/connect"
	"github.com/jmbenlloch/next_duck/pkg/database"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
)

const topiListProcedure = "/topi.daemon.v1.ConfigService/ListConfigurations"

func defaultTopiParams() database.Topiparam {
	return database.Topiparam{RabbitmqPort: 5672, RabbitmqVhost: "/", ExchangeName: "production", ControlQueue: "topi_daemon_control"}
}

func (s *DuckAPIServer) readTopiParams(ctx context.Context) (database.Topiparam, error) {
	p, err := s.queries.GetTopiParams(ctx)
	if err == sql.ErrNoRows {
		return defaultTopiParams(), nil
	}
	return p, err
}

func publicTopiConfiguration(p database.Topiparam) *pb.TopiConfiguration {
	return &pb.TopiConfiguration{
		Enabled: p.Enabled, DaemonUrl: p.DaemonUrl, RabbitmqAddress: p.RabbitmqAddress,
		RabbitmqPort: p.RabbitmqPort, RabbitmqUser: p.RabbitmqUser, RabbitmqVhost: p.RabbitmqVhost,
		ExchangeName: p.ExchangeName, ControlQueue: p.ControlQueue,
		SelectedConfiguration: p.SelectedConfiguration, HasApiToken: p.ApiToken != "",
		HasRabbitmqPassword: p.RabbitmqPassword != "",
	}
}

func (s *DuckAPIServer) GetTopiConfiguration(ctx context.Context, _ *connect.Request[pb.GetTopiConfigurationRequest]) (*connect.Response[pb.GetTopiConfigurationResponse], error) {
	p, err := s.readTopiParams(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("could not read TOPI configuration"))
	}
	return connect.NewResponse(&pb.GetTopiConfigurationResponse{Configuration: publicTopiConfiguration(p)}), nil
}

func validateTopiConfiguration(c *pb.TopiConfiguration, token, password string) error {
	if c == nil {
		return fmt.Errorf("configuration is required")
	}
	if c.RabbitmqPort < 1 || c.RabbitmqPort > 65535 {
		return fmt.Errorf("RabbitMQ port must be between 1 and 65535")
	}
	if strings.ContainsAny(c.SelectedConfiguration, `/\\`) {
		return fmt.Errorf("TOPI configuration must be a file name")
	}
	if !c.Enabled {
		return nil
	}
	u, err := url.Parse(c.DaemonUrl)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("TOPI daemon URL must be an HTTP(S) URL without credentials, query, or fragment")
	}
	if token == "" {
		return fmt.Errorf("TOPI API token is required when enabled")
	}
	if c.RabbitmqAddress == "" || c.RabbitmqUser == "" || password == "" || c.RabbitmqVhost == "" || c.ExchangeName == "" || c.ControlQueue == "" {
		return fmt.Errorf("all RabbitMQ fields and credentials are required when TOPI is enabled")
	}
	if c.SelectedConfiguration == "" {
		return fmt.Errorf("a TOPI configuration must be selected when enabled")
	}
	return nil
}

func (s *DuckAPIServer) UpdateTopiConfiguration(ctx context.Context, req *connect.Request[pb.UpdateTopiConfigurationRequest]) (*connect.Response[pb.UpdateTopiConfigurationResponse], error) {
	old, err := s.readTopiParams(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("could not read TOPI configuration"))
	}
	token, password := req.Msg.ApiToken, req.Msg.RabbitmqPassword
	if token == "" {
		token = old.ApiToken
	}
	if password == "" {
		password = old.RabbitmqPassword
	}
	if err := validateTopiConfiguration(req.Msg.Configuration, token, password); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	c := req.Msg.Configuration
	err = s.queries.UpsertTopiParams(ctx, database.UpsertTopiParamsParams{
		Enabled: c.Enabled, DaemonUrl: strings.TrimRight(c.DaemonUrl, "/"), ApiToken: token,
		RabbitmqAddress: c.RabbitmqAddress, RabbitmqPort: c.RabbitmqPort, RabbitmqUser: c.RabbitmqUser,
		RabbitmqPassword: password, RabbitmqVhost: c.RabbitmqVhost, ExchangeName: c.ExchangeName,
		ControlQueue: c.ControlQueue, SelectedConfiguration: c.SelectedConfiguration,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("could not store TOPI configuration"))
	}
	return connect.NewResponse(&pb.UpdateTopiConfigurationResponse{Success: true, Message: "TOPI configuration updated"}), nil
}

func (s *DuckAPIServer) ListTopiConfigurations(ctx context.Context, req *connect.Request[pb.ListTopiConfigurationsRequest]) (*connect.Response[pb.ListTopiConfigurationsResponse], error) {
	p, err := s.readTopiParams(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("could not read TOPI configuration"))
	}
	if req.Msg.DaemonUrl != "" {
		p.DaemonUrl = strings.TrimRight(req.Msg.DaemonUrl, "/")
	}
	if req.Msg.ApiToken != "" {
		p.ApiToken = req.Msg.ApiToken
	}
	u, parseErr := url.Parse(p.DaemonUrl)
	if parseErr != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid TOPI daemon URL"))
	}
	if p.ApiToken == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("TOPI daemon URL and token are not configured"))
	}
	body, _ := json.Marshal(struct{}{})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.DaemonUrl, "/")+topiListProcedure, bytes.NewReader(body))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid TOPI daemon URL"))
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.ApiToken)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Connect-Protocol-Version", "1")
	client := s.topiHTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("TOPI daemon is unavailable"))
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		io.Copy(io.Discard, resp.Body)
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("TOPI daemon returned HTTP %d", resp.StatusCode))
	}
	var result struct {
		Configurations []struct{ Name, ParseError string } `json:"configurations"`
		DefaultName    string                              `json:"defaultName"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&result); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("invalid response from TOPI daemon"))
	}
	out := make([]*pb.TopiAvailableConfiguration, 0, len(result.Configurations))
	for _, c := range result.Configurations {
		out = append(out, &pb.TopiAvailableConfiguration{Name: c.Name, ParseError: c.ParseError, IsDefault: c.Name == result.DefaultName})
	}
	return connect.NewResponse(&pb.ListTopiConfigurationsResponse{Configurations: out}), nil
}
