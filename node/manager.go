package node

import (
	"crypto/md5"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	MaxNodeID = 1023
)

var (
	ErrNodeIDTaken     = errors.New("node ID already taken")
	ErrMaxNodesReached = errors.New("maximum number of nodes reached")
	ErrInvalidNodeID   = errors.New("invalid node ID")
)

// NodeRegistry manages node ID assignments
type NodeRegistry struct {
	assignedNodes map[uint16]*NodeInfo
	nextNodeID    uint16
}

// NodeInfo contains information about a registered node
type NodeInfo struct {
	ID          uint16    `json:"id"`
	Hostname    string    `json:"hostname"`
	IPAddress   string    `json:"ip_address"`
	ProcessID   int       `json:"process_id"`
	StartTime   time.Time `json:"start_time"`
	LastSeen    time.Time `json:"last_seen"`
	IsActive    bool      `json:"is_active"`
}

// NewNodeRegistry creates a new node registry
func NewNodeRegistry() *NodeRegistry {
	return &NodeRegistry{
		assignedNodes: make(map[uint16]*NodeInfo),
		nextNodeID:    0,
	}
}

// RegisterNode registers a new node and returns its assigned ID
func (nr *NodeRegistry) RegisterNode(hostname, ipAddress string) (uint16, error) {
	// Check if we've reached the maximum number of nodes
	if len(nr.assignedNodes) >= MaxNodeID+1 {
		return 0, ErrMaxNodesReached
	}

	// Find next available node ID
	for nr.assignedNodes[nr.nextNodeID] != nil {
		nr.nextNodeID = (nr.nextNodeID + 1) % (MaxNodeID + 1)
	}

	nodeID := nr.nextNodeID
	nodeInfo := &NodeInfo{
		ID:        nodeID,
		Hostname:  hostname,
		IPAddress: ipAddress,
		ProcessID: os.Getpid(),
		StartTime: time.Now(),
		LastSeen:  time.Now(),
		IsActive:  true,
	}

	nr.assignedNodes[nodeID] = nodeInfo
	nr.nextNodeID = (nr.nextNodeID + 1) % (MaxNodeID + 1)

	return nodeID, nil
}

// UnregisterNode unregisters a node
func (nr *NodeRegistry) UnregisterNode(nodeID uint16) error {
	if nodeInfo, exists := nr.assignedNodes[nodeID]; exists {
		nodeInfo.IsActive = false
		nodeInfo.LastSeen = time.Now()
		return nil
	}
	return ErrInvalidNodeID
}

// GetNodeInfo returns information about a specific node
func (nr *NodeRegistry) GetNodeInfo(nodeID uint16) (*NodeInfo, bool) {
	nodeInfo, exists := nr.assignedNodes[nodeID]
	return nodeInfo, exists
}

// GetActiveNodes returns all active nodes
func (nr *NodeRegistry) GetActiveNodes() map[uint16]*NodeInfo {
	activeNodes := make(map[uint16]*NodeInfo)
	for id, node := range nr.assignedNodes {
		if node.IsActive {
			activeNodes[id] = node
		}
	}
	return activeNodes
}

// UpdateNodeHeartbeat updates the last seen time for a node
func (nr *NodeRegistry) UpdateNodeHeartbeat(nodeID uint16) error {
	if nodeInfo, exists := nr.assignedNodes[nodeID]; exists {
		nodeInfo.LastSeen = time.Now()
		return nil
	}
	return ErrInvalidNodeID
}

// AutoAssignNodeID automatically assigns a node ID based on hostname and IP
func AutoAssignNodeID() (uint16, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	ipAddress, err := getLocalIP()
	if err != nil {
		ipAddress = "127.0.0.1"
	}

	// Create a deterministic node ID based on hostname and IP
	nodeID := generateNodeIDFromIdentifiers(hostname, ipAddress)
	
	return nodeID, nil
}

// generateNodeIDFromIdentifiers generates a node ID from hostname and IP
func generateNodeIDFromIdentifiers(hostname, ipAddress string) uint16 {
	// Combine hostname and IP address
	identifier := fmt.Sprintf("%s:%s:%d", hostname, ipAddress, os.Getpid())
	
	// Create MD5 hash
	hash := md5.Sum([]byte(identifier))
	
	// Use first 2 bytes of hash to generate node ID
	nodeID := uint16(hash[0])<<8 | uint16(hash[1])
	
	// Ensure it's within the valid range (0-1023)
	return nodeID % (MaxNodeID + 1)
}

// getLocalIP gets the local IP address
func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// ValidateNodeID validates if a node ID is within valid range
func ValidateNodeID(nodeID uint16) error {
	if nodeID > MaxNodeID {
		return fmt.Errorf("%w: %d (max: %d)", ErrInvalidNodeID, nodeID, MaxNodeID)
	}
	return nil
}

// ParseNodeIDFromString parses node ID from string
func ParseNodeIDFromString(s string) (uint16, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(s), 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid node ID format: %w", err)
	}
	
	nodeID := uint16(id)
	if err := ValidateNodeID(nodeID); err != nil {
		return 0, err
	}
	
	return nodeID, nil
}

// ConfigurationService simulates a simple configuration service
type ConfigurationService struct {
	registry *NodeRegistry
}

// NewConfigurationService creates a new configuration service
func NewConfigurationService() *ConfigurationService {
	return &ConfigurationService{
		registry: NewNodeRegistry(),
	}
}

// RequestNodeID requests a node ID from the configuration service
func (cs *ConfigurationService) RequestNodeID() (uint16, error) {
	hostname, _ := os.Hostname()
	ip, _ := getLocalIP()
	
	return cs.registry.RegisterNode(hostname, ip)
}

// ReleaseNodeID releases a node ID back to the configuration service
func (cs *ConfigurationService) ReleaseNodeID(nodeID uint16) error {
	return cs.registry.UnregisterNode(nodeID)
}

// GetNodeRegistry returns the internal node registry (for testing/monitoring)
func (cs *ConfigurationService) GetNodeRegistry() *NodeRegistry {
	return cs.registry
}

// HealthCheck performs a health check on the configuration service
func (cs *ConfigurationService) HealthCheck() bool {
	// Simple health check - ensure registry is not nil and can assign nodes
	return cs.registry != nil && len(cs.registry.assignedNodes) <= MaxNodeID
} 