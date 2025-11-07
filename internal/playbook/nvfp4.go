package playbook

import (
	"fmt"
	"strings"
)

// runNVFP4 handles NVFP4 quantization commands
func (m *Manager) runNVFP4(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("nvfp4 command required. Usage: dgx run nvfp4 <quantize|setup>")
	}

	command := args[0]

	switch command {
	case "setup":
		return m.nvfp4Setup()
	case "quantize":
		if len(args) < 2 {
			return fmt.Errorf("model name required. Usage: dgx run nvfp4 quantize <model-name>")
		}
		return m.nvfp4Quantize(args[1])
	default:
		return fmt.Errorf("unknown nvfp4 command: %s", command)
	}
}

// nvfp4Setup prepares the environment for NVFP4 quantization
func (m *Manager) nvfp4Setup() error {
	fmt.Println("Setting up NVFP4 quantization environment...")

	// Create output directory
	fmt.Println("Creating output directory...")
	_, err := m.sshClient.Execute("mkdir -p ~/nvfp4_output")
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Pull TensorRT container
	fmt.Println("Pulling TensorRT container...")
	output, err := m.sshClient.Execute("docker pull nvcr.io/nvidia/tensorrt:25.12-py3")
	if err != nil {
		return fmt.Errorf("failed to pull container: %w", err)
	}

	fmt.Println(output)
	fmt.Println("\n✓ NVFP4 environment setup complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Set HF_TOKEN: export HF_TOKEN=your_token_here")
	fmt.Println("  2. Run quantization: dgx run nvfp4 quantize <model-name>")
	return nil
}

// nvfp4Quantize runs NVFP4 quantization on a model
func (m *Manager) nvfp4Quantize(modelName string) error {
	fmt.Printf("Starting NVFP4 quantization for model: %s\n", modelName)
	fmt.Println("This process may take 10-30 minutes depending on model size...")

	// Check if HF_TOKEN is set
	fmt.Println("\nChecking for Hugging Face token...")
	tokenCheck, _ := m.sshClient.Execute("echo $HF_TOKEN")
	if strings.TrimSpace(tokenCheck) == "" {
		fmt.Println("⚠️  Warning: HF_TOKEN not set")
		fmt.Println("Set it with: export HF_TOKEN=your_token_here")
		fmt.Println("Or run with: HF_TOKEN=xxx dgx run nvfp4 quantize ...")
	}

	// Build quantization command
	cmd := fmt.Sprintf(`docker run --rm \
		--gpus all \
		-v ~/nvfp4_output:/workspace/output \
		-e HF_TOKEN=$HF_TOKEN \
		nvcr.io/nvidia/tensorrt:25.12-py3 \
		bash -c "
			git clone https://github.com/NVIDIA/TensorRT-Model-Optimizer.git /tmp/trt-opt && \
			cd /tmp/trt-opt && \
			pip install -e . && \
			python examples/llm_ptq/hf_ptq.py \
				--model_name %s \
				--qformat fp4 \
				--output_dir /workspace/output
		"`, modelName)

	fmt.Println("\nStarting quantization...")
	fmt.Println("(This will stream output from the DGX)")

	output, err := m.sshClient.Execute(cmd)
	if err != nil {
		return fmt.Errorf("quantization failed: %w", err)
	}

	fmt.Println(output)
	fmt.Println("\n✓ NVFP4 quantization complete!")
	fmt.Printf("Output saved to: ~/nvfp4_output on DGX\n")
	fmt.Println("\nTo download the quantized model:")
	fmt.Println("  dgx sync dgx:~/nvfp4_output ./quantized_models")
	return nil
}
