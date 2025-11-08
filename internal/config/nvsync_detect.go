package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// NVSyncProfile captures connection details exported by the NVIDIA Sync app.
type NVSyncProfile struct {
	Host         string
	User         string
	Port         int
	IdentityFile string
	ConfigPath   string
}

// DetectNVSyncProfile returns the first NVIDIA Sync profile found on disk.
// It returns (nil, nil) when NVIDIA Sync is not installed/configured.
func DetectNVSyncProfile() (*NVSyncProfile, error) {
	profiles, err := detectNVSyncProfiles()
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, nil
	}
	return profiles[0], nil
}

func detectNVSyncProfiles() ([]*NVSyncProfile, error) {
	configPaths, err := nvSyncConfigPaths()
	if err != nil {
		return nil, err
	}

	var profiles []*NVSyncProfile
	for _, path := range configPaths {
		fileProfiles, err := parseNVSyncConfig(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		profiles = append(profiles, fileProfiles...)
	}

	return profiles, nil
}

func nvSyncConfigPaths() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine home directory: %w", err)
	}

	var paths []string
	add := func(path string) {
		if path != "" {
			paths = append(paths, filepath.Clean(path))
		}
	}

	if override := os.Getenv("NV_SYNC_SSH_CONFIG"); override != "" {
		add(override)
	}

	// macOS default install location
	add(filepath.Join(home, "Library", "Application Support", "NVIDIA", "Sync", "config", "ssh_config"))

	// Linux config locations (XDG + legacy)
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		add(filepath.Join(xdgConfig, "nvidia-sync", "ssh_config"))
		add(filepath.Join(xdgConfig, "NVIDIA", "Sync", "ssh_config"))
	}
	add(filepath.Join(home, ".config", "nvidia-sync", "ssh_config"))
	add(filepath.Join(home, ".config", "NVIDIA", "Sync", "ssh_config"))

	// Linux data locations (Ubuntu packages sometimes land here)
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		add(filepath.Join(xdgData, "nvidia-sync", "ssh_config"))
		add(filepath.Join(xdgData, "NVIDIA", "Sync", "ssh_config"))
	}
	add(filepath.Join(home, ".local", "share", "nvidia-sync", "ssh_config"))
	add(filepath.Join(home, ".local", "share", "NVIDIA", "Sync", "ssh_config"))

	// Windows (roaming/local AppData)
	if appData := os.Getenv("APPDATA"); appData != "" {
		add(filepath.Join(appData, "NVIDIA", "Sync", "config", "ssh_config"))
	} else {
		add(filepath.Join(home, "AppData", "Roaming", "NVIDIA", "Sync", "config", "ssh_config"))
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		add(filepath.Join(localAppData, "NVIDIA", "Sync", "config", "ssh_config"))
	} else {
		add(filepath.Join(home, "AppData", "Local", "NVIDIA", "Sync", "config", "ssh_config"))
	}

	return paths, nil
}

func parseNVSyncConfig(path string) ([]*NVSyncProfile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	profiles, err := parseNVSyncProfileReader(file)
	if err != nil {
		return nil, err
	}
	for _, profile := range profiles {
		profile.ConfigPath = path
	}
	return profiles, nil
}

func parseNVSyncProfileReader(r io.Reader) ([]*NVSyncProfile, error) {
	scanner := bufio.NewScanner(r)
	var (
		current  *NVSyncProfile
		profiles []*NVSyncProfile
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.ToLower(fields[0])
		value := strings.TrimSpace(strings.Join(fields[1:], " "))
		value = strings.Trim(value, "\"'")

		switch key {
		case "host":
			if current != nil {
				if finalized := finalizeNVSyncProfile(current); finalized != nil {
					profiles = append(profiles, finalized)
				}
			}
			alias := strings.Fields(value)
			if len(alias) == 0 || strings.Contains(alias[0], "*") {
				current = nil
				continue
			}
			current = &NVSyncProfile{Host: alias[0], Port: 22}
		case "hostname":
			if current != nil {
				current.Host = value
			}
		case "user":
			if current != nil {
				current.User = value
			}
		case "port":
			if current != nil {
				if p, err := strconv.Atoi(value); err == nil {
					current.Port = p
				}
			}
		case "identityfile":
			if current != nil {
				current.IdentityFile = normalizeNVSyncPath(value)
			}
		}
	}

	if current != nil {
		if finalized := finalizeNVSyncProfile(current); finalized != nil {
			profiles = append(profiles, finalized)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return profiles, nil
}

func finalizeNVSyncProfile(profile *NVSyncProfile) *NVSyncProfile {
	if profile == nil {
		return nil
	}

	if profile.Port == 0 {
		profile.Port = 22
	}

	if profile.Host == "" || profile.User == "" || profile.IdentityFile == "" {
		return nil
	}

	profile.IdentityFile = normalizeNVSyncPath(profile.IdentityFile)
	if _, err := os.Stat(profile.IdentityFile); err != nil {
		return nil
	}

	return profile
}

func normalizeNVSyncPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "\"'")
	path = os.ExpandEnv(path)
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			switch {
			case len(path) == 1:
				path = home
			case len(path) > 1 && path[1] == '/':
				path = filepath.Join(home, path[2:])
			}
		}
	}
	return filepath.Clean(path)
}
