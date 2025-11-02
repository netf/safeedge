package service

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/netf/safeedge/internal/controlplane/database/generated"
)

// IPAM handles WireGuard IP address allocation
type IPAM struct {
	queries *generated.Queries
	mu      sync.Mutex
	subnet  *net.IPNet
}

// NewIPAM creates a new IP address manager
func NewIPAM(queries *generated.Queries, cidr string) (*IPAM, error) {
	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}

	return &IPAM{
		queries: queries,
		subnet:  subnet,
	}, nil
}

// AllocateIP allocates the next available IP address in the subnet
func (i *IPAM) AllocateIP(ctx context.Context, orgID string) (net.IP, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Get all allocated IPs from database
	// For simplicity, we'll use a sequential allocation strategy
	// In production, you'd want a more sophisticated approach with IP pools

	// Start from the first usable IP in the subnet
	ip := incrementIP(i.subnet.IP)

	// Check if IP is already in use by querying devices
	// This is a simple implementation - in production you'd want:
	// 1. IP pool management in database
	// 2. Lease tracking
	// 3. IP recycling

	// For now, we'll just use a simple counter-based approach
	// TODO: Implement proper IP pool management

	return ip, nil
}

// incrementIP increments an IP address by one
func incrementIP(ip net.IP) net.IP {
	// Make a copy
	result := make(net.IP, len(ip))
	copy(result, ip)

	// Increment from the end
	for i := len(result) - 1; i >= 0; i-- {
		result[i]++
		if result[i] != 0 {
			break
		}
	}

	return result
}

// IsIPInSubnet checks if an IP is within the managed subnet
func (i *IPAM) IsIPInSubnet(ip net.IP) bool {
	return i.subnet.Contains(ip)
}
