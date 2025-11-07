package tunnel

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/weatherman/dgx-manager/pkg/types"
)

// Manager handles SSH tunnel management
type Manager struct {
	config *types.Config
}

// NewManager creates a new tunnel manager
func NewManager(config *types.Config) *Manager {
	return &Manager{
		config: config,
	}
}

// Create creates a new SSH tunnel in the background
func (m *Manager) Create(tunnel types.Tunnel) error {
	// Build SSH command for port forwarding
	args := []string{
		"-N", // Don't execute remote command
		"-f", // Go to background
		"-i", m.config.IdentityFile,
		"-p", fmt.Sprintf("%d", m.config.Port),
		"-L", fmt.Sprintf("%d:%s:%d", tunnel.LocalPort, tunnel.RemoteHost, tunnel.RemotePort),
		fmt.Sprintf("%s@%s", m.config.User, m.config.Host),
	}

	cmd := exec.Command("ssh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tunnel: %w", err)
	}

	// Find the PID of the SSH process we just created
	pid, err := m.findTunnelPID(tunnel.LocalPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not find tunnel PID: %v\n", err)
	} else {
		tunnel.PID = pid
	}

	tunnel.CreatedAt = time.Now()

	fmt.Printf("✓ Tunnel created: localhost:%d -> %s:%d (PID: %d)\n",
		tunnel.LocalPort, tunnel.RemoteHost, tunnel.RemotePort, tunnel.PID)

	return nil
}

// List returns all active SSH tunnels
func (m *Manager) List() ([]types.Tunnel, error) {
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	var activeTunnels []types.Tunnel
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if !strings.Contains(line, "ssh") || !strings.Contains(line, "-L") {
			continue
		}

		// Parse SSH command line to extract tunnel info
		tunnel, err := m.parseTunnelFromPS(line)
		if err != nil {
			continue
		}

		// Check if this tunnel is for our DGX host
		if strings.Contains(line, m.config.Host) {
			activeTunnels = append(activeTunnels, tunnel)
		}
	}

	return activeTunnels, nil
}

// Kill terminates a tunnel by PID
func (m *Manager) Kill(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to kill process %d: %w", pid, err)
	}

	fmt.Printf("✓ Tunnel (PID %d) terminated\n", pid)
	return nil
}

// KillAll terminates all tunnels to the DGX host
func (m *Manager) KillAll() error {
	tunnels, err := m.List()
	if err != nil {
		return err
	}

	for _, tunnel := range tunnels {
		if err := m.Kill(tunnel.PID); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to kill tunnel %d: %v\n", tunnel.PID, err)
		}
	}

	return nil
}

// findTunnelPID finds the PID of an SSH tunnel by local port
func (m *Manager) findTunnelPID(localPort int) (int, error) {
	// Use lsof to find the process listening on the local port
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", localPort))
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	pidStr := strings.TrimSpace(string(output))
	lines := strings.Split(pidStr, "\n")
	if len(lines) == 0 {
		return 0, fmt.Errorf("no process found on port %d", localPort)
	}

	// Return the first PID (SSH process)
	pid, err := strconv.Atoi(lines[0])
	if err != nil {
		return 0, fmt.Errorf("invalid PID: %s", lines[0])
	}

	return pid, nil
}

// parseTunnelFromPS extracts tunnel information from ps output
func (m *Manager) parseTunnelFromPS(line string) (types.Tunnel, error) {
	fields := strings.Fields(line)
	if len(fields) < 11 {
		return types.Tunnel{}, fmt.Errorf("invalid ps line")
	}

	// Get PID (usually second field)
	pid, err := strconv.Atoi(fields[1])
	if err != nil {
		return types.Tunnel{}, fmt.Errorf("invalid PID")
	}

	// Find the -L flag and parse the port forwarding spec
	var localPort, remotePort int
	var remoteHost string

	for i, field := range fields {
		if field == "-L" && i+1 < len(fields) {
			// Parse format: localPort:remoteHost:remotePort
			parts := strings.Split(fields[i+1], ":")
			if len(parts) == 3 {
				localPort, _ = strconv.Atoi(parts[0])
				remoteHost = parts[1]
				remotePort, _ = strconv.Atoi(parts[2])
			}
			break
		}
	}

	return types.Tunnel{
		PID:        pid,
		LocalPort:  localPort,
		RemotePort: remotePort,
		RemoteHost: remoteHost,
	}, nil
}

// IsPortInUse checks if a local port is already in use
func (m *Manager) IsPortInUse(port int) bool {
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", port))
	output, err := cmd.Output()
	return err == nil && len(output) > 0
}

// FindAvailablePort finds an available port starting from the given port
func (m *Manager) FindAvailablePort(startPort int) int {
	for port := startPort; port < startPort+100; port++ {
		if !m.IsPortInUse(port) {
			return port
		}
	}
	return 0
}
