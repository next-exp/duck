package main

import (
	"math"
	"net"
	"testing"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

func TestFluctuatingRateIsBoundedAndVariesAcrossTimeScales(t *testing.T) {
	minRate, maxRate := math.Inf(1), 0.0
	values := make(map[int]bool)
	for second := 0; second < 180; second++ {
		for _, noise := range []float64{-1, 0, 1} {
			rate := fluctuatingRate(50, time.Duration(second)*time.Second, noise)
			if rate < 7.5 || rate > 125 {
				t.Fatalf("unbounded target: %v", rate)
			}
			minRate = math.Min(minRate, rate)
			maxRate = math.Max(maxRate, rate)
			values[int(rate)] = true
		}
	}
	if len(values) < 30 || maxRate-minRate < 60 {
		t.Fatal("profile does not provide visible variation")
	}
	if _, err := NewGenerator(GeneratorConfig{RateProfile: "invalid"}); err == nil {
		t.Fatal("invalid profile accepted")
	}
}

func TestGeneratorRecoversWhenUDPReceiverRestarts(t *testing.T) {
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	address := listener.LocalAddr().(*net.UDPAddr)
	generator, err := NewGenerator(GeneratorConfig{Equipment: duck.Equipment{HostIP: "127.0.0.1", HostPort: address.Port}, RateHz: 200, PacketsPerEvent: 1, PacketSize: 64})
	if err != nil {
		t.Fatal(err)
	}
	if err := generator.Start(); err != nil {
		t.Fatal(err)
	}
	defer generator.Stop()
	buffer := make([]byte, 512)
	listener.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := listener.ReadFromUDP(buffer); err != nil {
		t.Fatal(err)
	}
	listener.Close()
	// Allow an ICMP port-unreachable to reach the connected sender.
	time.Sleep(100 * time.Millisecond)
	if generator.GetState() != SourceStateRunning {
		t.Fatal("receiver stop terminated the generator")
	}
	listener, err = net.ListenUDP("udp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	listener.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := listener.ReadFromUDP(buffer); err != nil {
		t.Fatalf("did not resume after receiver restart: %v", err)
	}
}
