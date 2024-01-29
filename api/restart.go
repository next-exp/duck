package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/melbahja/goph"
)

type ServerRole int

const (
	GDCRole ServerRole = iota
	LDCRole
)

// SSHClient interface for SSH operations (allows mocking in tests)
type SSHClient interface {
	Run(cmd string) ([]byte, error)
	Close() error
}

// sshClientCreator is a function that creates SSH clients
// This can be replaced in tests with a mock
var sshClientCreator = func(user, host string) (SSHClient, error) {
	return NewSshClient(user, host)
}

func restartServices(configFilename string) error {
	configuration, err := duck.ReadConfiguration(configFilename, nil, nil)
	if err != nil {
		message := fmt.Errorf("could not read configuration file: %w", err)
		return message
	}
	enabledLDCs := duck.EnabledLDCs(configuration.LDCs)
	enabledGDCs := duck.EnabledGDCs(configuration.GDCs)

	var wg sync.WaitGroup

	for _, gdc := range enabledGDCs {
		wg.Add(1)
		go func(name, host string) {
			defer wg.Done()
			restartService(name, GDCRole, host)
		}(gdc.Name, gdc.Host)
	}

	for _, ldc := range enabledLDCs {
		wg.Add(1)
		go func(name, host string) {
			defer wg.Done()
			restartService(name, LDCRole, host)
		}(ldc.Name, ldc.Host)
	}

	wg.Wait()
	return nil
}

func restartService(name string, role ServerRole, host string) {
	message := fmt.Sprintf("Restarting services on %s", name)
	logger.Slog.Info(message)
	sshClient, err := sshClientCreator("dateuser", host)
	if err != nil {
		message := fmt.Errorf("could not restart services on %s: %w", name, err)
		logger.Slog.Error(message.Error())
		return
	}
	defer sshClient.Close()

	err = restartDuckSSH(sshClient, name, role)
	if err != nil {
		// Error already logged in restartDuckSSH
		return
	}
}

func restartDuckSSH(sshClient SSHClient, name string, role ServerRole) error {
	// Running sudo over ssh raises an error:
	// sudo: sorry, you must have a tty to run sudo
	// To fix this, comment the following line in /etc/sudoers:
	// #Defaults    requiretty
	var cmd string
	switch role {
	case GDCRole:
		cmd = "/home/dateuser/duck/restart_gdc.sh"
	case LDCRole:
		cmd = "/home/dateuser/duck/restart_ldc.sh"
	}
	out, err := sshClient.Run(cmd)
	if err != nil {
		message := fmt.Errorf("could not restart %s services: %w", name, err)
		logger.Slog.Error(message.Error())
		return err
	}
	output := string(out)
	output = strings.ReplaceAll(output, "[31m", "")
	output = strings.ReplaceAll(output, "[32m", "")
	output = strings.ReplaceAll(output, "[39m", "")

	message := fmt.Sprintf("%s:\n %s", name, output)
	logger.Slog.Info(message)
	return nil
}

func NewSshClient(user string, host string) (*goph.Client, error) {
	// Start new ssh connection with private key.
	keyPath := fmt.Sprintf("%s/.ssh/id_rsa", os.Getenv("HOME"))
	auth, err := goph.Key(keyPath, "")
	if err != nil {
		return nil, err
	}

	client, err := goph.New(user, host, auth)

	// Defer closing the network connection.
	//	defer client.Close()
	return client, err
}
