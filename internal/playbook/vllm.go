package playbook

import (
	"fmt"
	"strings"
)

// runVLLM handles vLLM playbook commands
func (m *Manager) runVLLM(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("vllm command required. Usage: dgx run vllm <pull|serve|status>")
	}

	command := args[0]

	switch command {
	case "pull":
		return m.vllmPull()
	case "serve":
		if len(args) < 2 {
			return fmt.Errorf("model name required. Usage: dgx run vllm serve <model>")
		}
		return m.vllmServe(args[1])
	case "status":
		return m.vllmStatus()
	case "stop":
		return m.vllmStop()
	default:
		return fmt.Errorf("unknown vllm command: %s", command)
	}
}

// vllmPull pulls the vLLM Docker container
func (m *Manager) vllmPull() error {
	fmt.Println("Pulling vLLM container...")
	fmt.Println("Image: nvcr.io/nvidia/vllm:25.09-py3")

	output, err := m.sshClient.Execute("docker pull nvcr.io/nvidia/vllm:25.09-py3")
	if err != nil {
		return fmt.Errorf("failed to pull container: %w", err)
	}

	fmt.Println(output)
	fmt.Println("\n✓ vLLM container pulled successfully!")
	return nil
}

// vllmServe starts a vLLM server with the specified model
func (m *Manager) vllmServe(model string) error {
	fmt.Printf("Starting vLLM server with model: %s\n", model)
	fmt.Println("This will run the server in a Docker container...")

	// Build the Docker run command
	cmd := fmt.Sprintf(`docker run -d \
		--name vllm-server \
		--gpus all \
		--shm-size=10g \
		-p 8000:8000 \
		nvcr.io/nvidia/vllm:25.09-py3 \
		vllm serve %s \
		--host 0.0.0.0 \
		--port 8000`, model)

	output, err := m.sshClient.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to start vLLM server: %w", err)
	}

	containerID := strings.TrimSpace(output)
	fmt.Printf("✓ vLLM server started (Container: %s)\n", containerID[:12])
	fmt.Println("\nTo access the API:")
	fmt.Println("  1. Create a tunnel: dgx tunnel create 8000:8000 \"vLLM\"")
	fmt.Println("  2. API endpoint: http://localhost:8000/v1")
	fmt.Println("\nTo check logs:")
	fmt.Println("  dgx exec docker logs -f vllm-server")
	return nil
}

// vllmStatus checks if vLLM is running
func (m *Manager) vllmStatus() error {
	fmt.Println("Checking vLLM status...")

	output, err := m.sshClient.Execute("docker ps --filter name=vllm-server --format '{{.ID}} {{.Status}} {{.Names}}'")
	if err != nil {
		return fmt.Errorf("failed to check status: %w", err)
	}

	if output == "" {
		fmt.Println("✗ vLLM server is not running")
		fmt.Println("\nTo start vLLM:")
		fmt.Println("  dgx run vllm serve <model-name>")
		return nil
	}

	fmt.Printf("✓ vLLM server is running\n%s\n", output)

	// Try to get health status
	healthCmd := "docker exec vllm-server curl -s http://localhost:8000/health || echo 'Not accessible'"
	health, _ := m.sshClient.Execute(healthCmd)
	if health != "" {
		fmt.Printf("\nHealth check: %s\n", strings.TrimSpace(health))
	}

	return nil
}

// vllmStop stops the vLLM server
func (m *Manager) vllmStop() error {
	fmt.Println("Stopping vLLM server...")

	output, err := m.sshClient.Execute("docker stop vllm-server && docker rm vllm-server")
	if err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	fmt.Println(output)
	fmt.Println("✓ vLLM server stopped and removed")
	return nil
}
