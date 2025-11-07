package playbook

import (
	"fmt"
	"strings"
)

// runOllama handles Ollama playbook commands
func (m *Manager) runOllama(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("ollama command required. Usage: dgx run ollama <install|pull|list|serve|status>")
	}

	command := args[0]

	switch command {
	case "install":
		return m.ollamaInstall()
	case "pull":
		if len(args) < 2 {
			return fmt.Errorf("model name required. Usage: dgx run ollama pull <model>")
		}
		return m.ollamaPull(args[1])
	case "list":
		return m.ollamaList()
	case "serve":
		return m.ollamaServe()
	case "status":
		return m.ollamaStatus()
	case "run":
		if len(args) < 2 {
			return fmt.Errorf("model name required. Usage: dgx run ollama run <model> [prompt]")
		}
		prompt := ""
		if len(args) > 2 {
			prompt = strings.Join(args[2:], " ")
		}
		return m.ollamaRun(args[1], prompt)
	default:
		return fmt.Errorf("unknown ollama command: %s", command)
	}
}

// ollamaInstall installs Ollama on the DGX
func (m *Manager) ollamaInstall() error {
	fmt.Println("Installing Ollama on DGX...")
	fmt.Println("Running: curl -fsSL https://ollama.com/install.sh | sh")

	output, err := m.sshClient.Execute("curl -fsSL https://ollama.com/install.sh | sh")
	if err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Println(output)
	fmt.Println("\n✓ Ollama installed successfully!")
	return nil
}

// ollamaPull downloads a model
func (m *Manager) ollamaPull(model string) error {
	fmt.Printf("Pulling model: %s...\n", model)

	output, err := m.sshClient.Execute(fmt.Sprintf("ollama pull %s", model))
	if err != nil {
		return fmt.Errorf("failed to pull model: %w", err)
	}

	fmt.Println(output)
	fmt.Printf("\n✓ Model %s downloaded successfully!\n", model)
	return nil
}

// ollamaList lists available models
func (m *Manager) ollamaList() error {
	fmt.Println("Available models on DGX:")

	output, err := m.sshClient.Execute("ollama list")
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	fmt.Println(output)
	return nil
}

// ollamaServe starts the Ollama service
func (m *Manager) ollamaServe() error {
	fmt.Println("Starting Ollama service...")
	fmt.Println("Note: This will run in the background on your DGX")

	output, err := m.sshClient.Execute("nohup ollama serve > /tmp/ollama.log 2>&1 & echo $!")
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	pid := strings.TrimSpace(output)
	fmt.Printf("✓ Ollama service started (PID: %s)\n", pid)
	fmt.Println("\nTo access Ollama API:")
	fmt.Println("  1. Create a tunnel: dgx tunnel create 11434:11434 \"Ollama\"")
	fmt.Println("  2. Access at: http://localhost:11434")
	return nil
}

// ollamaStatus checks if Ollama is running
func (m *Manager) ollamaStatus() error {
	fmt.Println("Checking Ollama status...")

	output, err := m.sshClient.Execute("pgrep -f 'ollama serve'")
	if err != nil || output == "" {
		fmt.Println("✗ Ollama is not running")
		fmt.Println("\nTo start Ollama:")
		fmt.Println("  dgx run ollama serve")
		return nil
	}

	pids := strings.TrimSpace(output)
	fmt.Printf("✓ Ollama is running (PID: %s)\n", pids)

	// Try to get version
	version, err := m.sshClient.Execute("ollama --version")
	if err == nil {
		fmt.Printf("Version: %s\n", strings.TrimSpace(version))
	}

	return nil
}

// ollamaRun runs a model with an optional prompt
func (m *Manager) ollamaRun(model string, prompt string) error {
	if prompt == "" {
		// Interactive mode - not supported via Execute, suggest connect
		fmt.Printf("Interactive mode is not supported via 'dgx run'.\n")
		fmt.Println("\nTo run Ollama interactively:")
		fmt.Println("  1. Connect to your DGX: dgx connect")
		fmt.Printf("  2. Run: ollama run %s\n", model)
		return nil
	}

	// Single prompt mode
	fmt.Printf("Running %s with prompt...\n", model)

	cmd := fmt.Sprintf("ollama run %s '%s'", model, prompt)
	output, err := m.sshClient.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to run model: %w", err)
	}

	fmt.Println("\nResponse:")
	fmt.Println(output)
	return nil
}
