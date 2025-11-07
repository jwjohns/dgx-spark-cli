package types

import "time"

// Config represents the DGX connection configuration
type Config struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"user"`
	IdentityFile string `yaml:"identity_file"`
	Tunnels      []Tunnel `yaml:"tunnels,omitempty"`
}

// Tunnel represents an SSH tunnel configuration
type Tunnel struct {
	ID          string    `yaml:"id"`
	LocalPort   int       `yaml:"local_port"`
	RemotePort  int       `yaml:"remote_port"`
	RemoteHost  string    `yaml:"remote_host"` // Usually "localhost"
	Description string    `yaml:"description,omitempty"`
	PID         int       `yaml:"-"` // Process ID, not saved to config
	CreatedAt   time.Time `yaml:"created_at,omitempty"`
}

// GPUInfo represents GPU status information
type GPUInfo struct {
	ID          int
	Name        string
	MemoryUsed  string
	MemoryTotal string
	Utilization string
	Temperature string
	Processes   []GPUProcess
}

// GPUProcess represents a process using the GPU
type GPUProcess struct {
	PID         int
	Name        string
	MemoryUsage string
}

// ConnectionStatus represents the current connection state
type ConnectionStatus struct {
	Connected     bool
	Host          string
	ActiveTunnels int
	Latency       time.Duration
}
