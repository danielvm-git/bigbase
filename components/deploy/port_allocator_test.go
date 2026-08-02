package deploy

import (
	"fmt"
	"net"
	"testing"
)

func TestGetFreePort_ReturnsValidPort(t *testing.T) {
	port, err := GetFreePort()
	if err != nil {
		t.Fatalf("GetFreePort failed: %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Fatalf("GetFreePort returned invalid port number: %d", port)
	}

	// Verify the returned port is actually free and can be listened on
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("GetFreePort returned port %d which could not be listened on: %v", port, err)
	}
	_ = ln.Close()
}

func TestGetFreePort_MultipleAllocationsUnique(t *testing.T) {
	ports := make(map[int]bool)
	for i := 0; i < 10; i++ {
		port, err := GetFreePort()
		if err != nil {
			t.Fatalf("GetFreePort attempt %d failed: %v", i, err)
		}
		if ports[port] {
			t.Errorf("GetFreePort returned duplicate port %d on attempt %d", port, i)
		}
		ports[port] = true
	}
}

func TestPickPort_DynamicWhenBaseZero(t *testing.T) {
	port, err := pickPort(0)
	if err != nil {
		t.Fatalf("pickPort(0) failed: %v", err)
	}
	if port <= 0 {
		t.Fatalf("pickPort(0) returned invalid port: %d", port)
	}
}
