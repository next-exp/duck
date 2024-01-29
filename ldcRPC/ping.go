package main

import (
	"fmt"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
	probing "github.com/prometheus-community/pro-bing"
)

// This library attempts to send an "unprivileged" ping via UDP.
// On Linux, this must be enabled with the following sysctl command:
// sysctl -w net.ipv4.ping_group_range="0 2147483647"
// sudo setcap cap_net_raw+ep /path/to/your/binary

// Pinger abstracts ping operations for testing
type Pinger interface {
	SetCount(int)
	SetTimeout(time.Duration)
	Run() error
	PacketsReceived() int
}

// PingerFactory creates pingers for specific IPs
type PingerFactory interface {
	NewPinger(ip string) (Pinger, error)
}

// ProbingPinger wraps pro-bing.Pinger to implement Pinger interface
type ProbingPinger struct {
	*probing.Pinger
}

// SetCount sets the number of ping packets to send
func (p *ProbingPinger) SetCount(count int) {
	p.Pinger.Count = count
}

// SetTimeout sets the timeout for ping operations
func (p *ProbingPinger) SetTimeout(timeout time.Duration) {
	p.Pinger.Timeout = timeout
}

// PacketsReceived returns the number of packets received
func (p *ProbingPinger) PacketsReceived() int {
	stats := p.Pinger.Statistics()
	return stats.PacketsRecv
}

// ProbingFactory creates real pro-bing pingers
type ProbingFactory struct{}

// NewPinger creates a new pinger for the given IP address
func (f *ProbingFactory) NewPinger(ip string) (Pinger, error) {
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		return nil, err
	}
	return &ProbingPinger{Pinger: pinger}, nil
}

func pingDevices(s *server, factory PingerFactory, equipments []duck.Equipment) bool {
	for i := 0; i < len(equipments); i++ {
		equipment := equipments[i]
		if equipment.Enabled {
			pinger, err := factory.NewPinger(equipment.DeviceIP)
			if err != nil {
				s.logger.Slog.Error(err.Error())
				return false
			}
			pinger.SetCount(2)
			pinger.SetTimeout(500 * time.Millisecond)
			err = pinger.Run()
			if err != nil {
				s.logger.Slog.Error(err.Error())
				return false
			}
			packetsRecv := pinger.PacketsReceived()
			if packetsRecv < 1 {
				message := fmt.Sprintf("Device %s did not respond to ping request", equipment.DeviceIP)
				s.logger.Slog.Error(message)
				return false
			}
		}
	}
	return true
}
