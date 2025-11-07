package playbook

import (
	"fmt"

	"github.com/weatherman/dgx-manager/internal/ssh"
)

// Playbook represents a DGX Spark workflow
type Playbook struct {
	Name        string
	Description string
	Category    string
	Commands    []string
}

// Manager handles DGX Spark playbook execution
type Manager struct {
	sshClient *ssh.Client
}

// NewManager creates a new playbook manager
func NewManager(client *ssh.Client) *Manager {
	return &Manager{
		sshClient: client,
	}
}

// Available playbook categories
const (
	CategoryInference   = "Inference & Serving"
	CategoryFineTuning  = "Fine-tuning & Training"
	CategoryDevelopment = "Development Tools"
	CategoryNetworking  = "Networking"
	CategoryAdvanced    = "Advanced Applications"
)

// GetAvailablePlaybooks returns a list of all available playbooks
func GetAvailablePlaybooks() []Playbook {
	return []Playbook{
		// Inference & Serving
		{
			Name:        "ollama",
			Description: "Lightweight local model runner",
			Category:    CategoryInference,
		},
		{
			Name:        "vllm",
			Description: "Optimized LLM inference engine",
			Category:    CategoryInference,
		},
		{
			Name:        "trt-llm",
			Description: "TensorRT LLM for efficient inference",
			Category:    CategoryInference,
		},
		{
			Name:        "nim",
			Description: "NVIDIA Inference Microservices",
			Category:    CategoryInference,
		},
		{
			Name:        "speculative-decoding",
			Description: "Faster inference with speculative decoding",
			Category:    CategoryInference,
		},

		// Fine-tuning & Training
		{
			Name:        "nvfp4",
			Description: "4-bit FP quantization for Blackwell GPUs",
			Category:    CategoryFineTuning,
		},
		{
			Name:        "llama-factory",
			Description: "LLaMA model fine-tuning toolkit",
			Category:    CategoryFineTuning,
		},
		{
			Name:        "unsloth",
			Description: "Fast fine-tuning optimization",
			Category:    CategoryFineTuning,
		},
		{
			Name:        "nemo",
			Description: "NVIDIA NeMo fine-tuning framework",
			Category:    CategoryFineTuning,
		},

		// Development Tools
		{
			Name:        "vscode",
			Description: "VS Code setup for DGX Spark",
			Category:    CategoryDevelopment,
		},
		{
			Name:        "jupyter",
			Description: "JupyterLab environment",
			Category:    CategoryDevelopment,
		},
		{
			Name:        "comfyui",
			Description: "Node-based image generation UI",
			Category:    CategoryDevelopment,
		},
		{
			Name:        "open-webui",
			Description: "Web interface for local models",
			Category:    CategoryDevelopment,
		},
	}
}

// GetPlaybooksByCategory returns playbooks filtered by category
func GetPlaybooksByCategory(category string) []Playbook {
	all := GetAvailablePlaybooks()
	filtered := make([]Playbook, 0)

	for _, p := range all {
		if p.Category == category {
			filtered = append(filtered, p)
		}
	}

	return filtered
}

// GetPlaybook returns a specific playbook by name
func GetPlaybook(name string) (*Playbook, error) {
	for _, p := range GetAvailablePlaybooks() {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("playbook not found: %s", name)
}

// Execute runs a playbook command on the DGX
func (m *Manager) Execute(playbookName string, args []string) error {
	playbook, err := GetPlaybook(playbookName)
	if err != nil {
		return err
	}

	switch playbookName {
	case "ollama":
		return m.runOllama(args)
	case "vllm":
		return m.runVLLM(args)
	case "nvfp4":
		return m.runNVFP4(args)
	default:
		return fmt.Errorf("playbook '%s' is not yet implemented", playbook.Name)
	}
}
