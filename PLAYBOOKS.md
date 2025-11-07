# DGX Spark Playbooks

DGX Manager includes integrated support for [NVIDIA DGX Spark Playbooks](https://github.com/NVIDIA/dgx-spark-playbooks), making it easy to run AI/ML workloads on your DGX Spark.

## Available Playbooks

View all available playbooks:
```bash
dgx playbook list
```

## Quick Start

### Ollama - Local Model Runner

**Install Ollama:**
```bash
dgx run ollama install
```

**Pull a model:**
```bash
dgx run ollama pull qwen2.5:32b
dgx run ollama pull llama3.2:3b
```

**List downloaded models:**
```bash
dgx run ollama list
```

**Start Ollama service:**
```bash
dgx run ollama serve
```

**Check status:**
```bash
dgx run ollama status
```

**Run a model with a prompt:**
```bash
dgx run ollama run qwen2.5:32b "Explain quantum computing"
```

**Access via API:**
1. Start the service: `dgx run ollama serve`
2. Create tunnel: `dgx tunnel create 11434:11434 "Ollama"`
3. Use API: `curl http://localhost:11434/api/generate`

### vLLM - Optimized LLM Inference

**Pull vLLM container:**
```bash
dgx run vllm pull
```

**Serve a model:**
```bash
dgx run vllm serve meta-llama/Llama-2-7b-hf
dgx run vllm serve mistralai/Mistral-7B-v0.1
```

**Check status:**
```bash
dgx run vllm status
```

**Stop the server:**
```bash
dgx run vllm stop
```

**Access via API:**
1. After starting with `serve`, create tunnel: `dgx tunnel create 8000:8000 "vLLM"`
2. API endpoint: `http://localhost:8000/v1`
3. OpenAI-compatible: Use with any OpenAI SDK

### NVFP4 - 4-bit Quantization

**Setup environment:**
```bash
dgx run nvfp4 setup
```

**Quantize a model:**
```bash
# Set your Hugging Face token first
export HF_TOKEN=hf_your_token_here

# Run quantization
dgx run nvfp4 quantize meta-llama/Llama-2-7b-hf
dgx run nvfp4 quantize mistralai/Mistral-7B-v0.1
```

**Download quantized model:**
```bash
dgx sync dgx:~/nvfp4_output ./quantized_models
```

## Workflow Examples

### Complete Ollama Setup

```bash
# 1. Install Ollama
dgx run ollama install

# 2. Pull your favorite models
dgx run ollama pull qwen2.5:32b
dgx run ollama pull llama3.2:3b

# 3. Start the service
dgx run ollama serve

# 4. Create tunnel for API access
dgx tunnel create 11434:11434 "Ollama"

# 5. Use from your local machine
curl http://localhost:11434/api/generate -d '{
  "model": "qwen2.5:32b",
  "prompt": "Why is the sky blue?"
}'
```

### vLLM Production Deployment

```bash
# 1. Pull container
dgx run vllm pull

# 2. Start serving a model
dgx run vllm serve meta-llama/Llama-2-7b-hf

# 3. Create tunnel
dgx tunnel create 8000:8000 "vLLM API"

# 4. Test with OpenAI-compatible client
curl http://localhost:8000/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "meta-llama/Llama-2-7b-hf",
    "prompt": "San Francisco is a",
    "max_tokens": 50
  }'
```

### Model Quantization Pipeline

```bash
# 1. Setup quantization environment
dgx run nvfp4 setup

# 2. Check GPU status
dgx gpu

# 3. Run quantization (takes 10-30 minutes)
export HF_TOKEN=your_token
dgx run nvfp4 quantize meta-llama/Llama-2-7b-hf

# 4. Monitor progress
dgx exec docker logs -f $(docker ps -q --filter name=tensorrt)

# 5. Download results
dgx sync dgx:~/nvfp4_output ./quantized_models
```

## Advanced Usage

### Execute Custom Commands

Run any command on your DGX:
```bash
dgx exec nvidia-smi
dgx exec docker ps
dgx exec "cat /proc/cpuinfo | grep 'model name'"
```

### Combining Commands

```bash
# Start Ollama and vLLM simultaneously
dgx run ollama serve
dgx run vllm serve mistralai/Mistral-7B-v0.1

# Create tunnels for both
dgx tunnel create 11434:11434 "Ollama"
dgx tunnel create 8000:8000 "vLLM"

# Check both are running
dgx tunnel list
```

## Playbook Categories

### Inference & Serving
- **ollama** - Lightweight local models
- **vllm** - High-performance inference
- **trt-llm** - TensorRT LLM optimization
- **nim** - NVIDIA Inference Microservices
- **speculative-decoding** - Faster inference

### Fine-tuning & Training
- **nvfp4** - 4-bit quantization
- **llama-factory** - LLaMA fine-tuning
- **unsloth** - Fast optimization
- **nemo** - NVIDIA NeMo framework

### Development Tools
- **vscode** - VS Code setup
- **jupyter** - JupyterLab
- **comfyui** - Image generation
- **open-webui** - Web interface

## Tips

### Model Selection
- **Small models (3-7B)**: Great for testing, run on Ollama
- **Medium models (7-13B)**: Use vLLM for production
- **Large models (30B+)**: Quantize with NVFP4 first

### Performance
- Monitor GPU usage: `dgx gpu`
- Check running processes: `dgx exec docker ps`
- View logs: `dgx exec docker logs <container>`

### Troubleshooting
- If Ollama serve fails, check if port is in use
- For vLLM issues, verify GPU availability with `dgx gpu`
- NVFP4 requires HF_TOKEN environment variable

## More Information

- **Official Playbooks**: https://github.com/NVIDIA/dgx-spark-playbooks
- **DGX Manager Docs**: [README.md](README.md)
- **Getting Help**: `dgx run <playbook> --help`
