package node

import (
	"fmt"
	"sync"
	"testing"
)

func TestNewNodeRegistry(t *testing.T) {
	registry := NewNodeRegistry()

	if registry == nil {
		t.Fatal("NewNodeRegistry() returned nil")
	}

	if registry.assignedNodes == nil {
		t.Error("assignedNodes map should be initialized")
	}

	if len(registry.assignedNodes) != 0 {
		t.Error("assignedNodes should be empty initially")
	}
}

func TestRegisterNode(t *testing.T) {
	registry := NewNodeRegistry()

	hostname := "test-host"
	ipAddress := "192.168.1.1"

	nodeID, err := registry.RegisterNode(hostname, ipAddress)
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	if nodeID > MaxNodeID {
		t.Errorf("Node ID %d exceeds maximum %d", nodeID, MaxNodeID)
	}

	// Verify node info
	nodeInfo, exists := registry.GetNodeInfo(nodeID)
	if !exists {
		t.Error("Registered node should exist")
	}

	if nodeInfo.Hostname != hostname {
		t.Errorf("Expected hostname %s, got %s", hostname, nodeInfo.Hostname)
	}

	if nodeInfo.IPAddress != ipAddress {
		t.Errorf("Expected IP address %s, got %s", ipAddress, nodeInfo.IPAddress)
	}

	if !nodeInfo.IsActive {
		t.Error("Newly registered node should be active")
	}
}

func TestValidateNodeID(t *testing.T) {
	testCases := []struct {
		nodeID uint16
		valid  bool
	}{
		{0, true},
		{1, true},
		{MaxNodeID, true},
		{MaxNodeID + 1, false},
		{65535, false},
	}

	for _, tc := range testCases {
		err := ValidateNodeID(tc.nodeID)
		if tc.valid && err != nil {
			t.Errorf("Node ID %d should be valid, got error: %v", tc.nodeID, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("Node ID %d should be invalid", tc.nodeID)
		}
	}
}

func TestAutoAssignNodeID(t *testing.T) {
	nodeID, err := AutoAssignNodeID()
	if err != nil {
		t.Fatalf("AutoAssignNodeID failed: %v", err)
	}

	if nodeID > MaxNodeID {
		t.Errorf("Auto-assigned node ID %d exceeds maximum %d", nodeID, MaxNodeID)
	}
}

func TestGenerateSecureNodeID(t *testing.T) {
	hostname := "test-host"
	ipAddress := "192.168.1.1"

	// Generate multiple IDs and check they're within range
	for i := 0; i < 10; i++ {
		nodeID := generateSecureNodeID(hostname, ipAddress)
		if nodeID > MaxNodeID {
			t.Errorf("Generated node ID %d exceeds maximum %d", nodeID, MaxNodeID)
		}
	}
}

func TestConcurrentRegistration(t *testing.T) {
	registry := NewNodeRegistry()

	const numGoroutines = 10
	var wg sync.WaitGroup
	nodeIDs := make(chan uint16, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Concurrent node registration
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			hostname := fmt.Sprintf("host-%d", index)
			ipAddress := fmt.Sprintf("192.168.1.%d", index%254+1)

			nodeID, err := registry.RegisterNode(hostname, ipAddress)
			if err != nil {
				errors <- err
			} else {
				nodeIDs <- nodeID
			}
		}(i)
	}

	wg.Wait()
	close(nodeIDs)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent registration error: %v", err)
	}

	// Check for duplicate node IDs
	seenIDs := make(map[uint16]bool)
	count := 0
	for nodeID := range nodeIDs {
		if seenIDs[nodeID] {
			t.Errorf("Duplicate node ID assigned: %d", nodeID)
		}
		seenIDs[nodeID] = true
		count++
	}

	if count != numGoroutines {
		t.Errorf("Expected %d successful registrations, got %d", numGoroutines, count)
	}
}
