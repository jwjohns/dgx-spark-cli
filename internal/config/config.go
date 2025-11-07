package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/weatherman/dgx-manager/pkg/types"
	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigDir  = ".config/dgx"
	DefaultConfigFile = "config.yaml"
)

// Manager handles configuration persistence
type Manager struct {
	configPath string
	config     *types.Config
}

// NewManager creates a new configuration manager
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, DefaultConfigDir)
	configPath := filepath.Join(configDir, DefaultConfigFile)

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	m := &Manager{
		configPath: configPath,
	}

	// Load existing config or create default
	if err := m.Load(); err != nil {
		if os.IsNotExist(err) {
			m.config = m.defaultConfig()
			if err := m.Save(); err != nil {
				return nil, fmt.Errorf("failed to save default config: %w", err)
			}
		} else {
			return nil, err
		}
	}

	return m, nil
}

// Load reads the configuration from disk
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	var cfg types.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	m.config = &cfg
	return nil
}

// Save writes the configuration to disk
func (m *Manager) Save() error {
	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Get returns the current configuration
func (m *Manager) Get() *types.Config {
	return m.config
}

// Set updates the configuration
func (m *Manager) Set(cfg *types.Config) error {
	m.config = cfg
	return m.Save()
}

// Update updates specific fields and saves
func (m *Manager) Update(updateFn func(*types.Config)) error {
	updateFn(m.config)
	return m.Save()
}

// AddTunnel adds a tunnel to the configuration
func (m *Manager) AddTunnel(tunnel types.Tunnel) error {
	m.config.Tunnels = append(m.config.Tunnels, tunnel)
	return m.Save()
}

// RemoveTunnel removes a tunnel from the configuration by ID
func (m *Manager) RemoveTunnel(id string) error {
	tunnels := make([]types.Tunnel, 0)
	for _, t := range m.config.Tunnels {
		if t.ID != id {
			tunnels = append(tunnels, t)
		}
	}
	m.config.Tunnels = tunnels
	return m.Save()
}

// GetTunnel retrieves a tunnel by ID
func (m *Manager) GetTunnel(id string) (*types.Tunnel, error) {
	for _, t := range m.config.Tunnels {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("tunnel not found: %s", id)
}

// defaultConfig returns a default configuration
func (m *Manager) defaultConfig() *types.Config {
	home, _ := os.UserHomeDir()

	return &types.Config{
		Host:         "", // User must configure
		Port:         22,
		User:         "",
		IdentityFile: filepath.Join(home, ".ssh", "id_ed25519"),
		Tunnels:      []types.Tunnel{},
	}
}

// GetConfigPath returns the path to the config file
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// IsConfigured checks if the essential configuration is set
func (m *Manager) IsConfigured() bool {
	return m.config.Host != "" && m.config.User != ""
}
