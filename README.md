# DGX Manager

A powerful CLI tool to manage connections, SSH tunnels, GPU monitoring, and AI/ML workloads for your DGX Spark system.

## Features

- **SSH Connection Management** - Quick access to your DGX Spark
- **Dynamic Port Forwarding** - Create and manage SSH tunnels on the fly
- **GPU Monitoring** - Real-time GPU status, memory usage, and process tracking
- **File Synchronization** - Easy rsync-based file transfers
- **Configuration Management** - Persistent connection settings
- **Integrated Playbooks** - Run Ollama, vLLM, NVFP4 quantization, and more with simple commands

## Installation

### Prerequisites

- Go 1.24+ (for building from source)
- SSH client
- rsync (for file sync)
- Task (optional, for build automation)

### Build from Source

```bash
# Clone the repository
git clone git@github.com:jwjohns/dgx-spark-cli.git
cd dgx-spark-cli

# Option 1: Using the install script (recommended)
./install.sh

# Option 2: Manual install to ~/.local/bin
go build -o dgx ./cmd/dgx
mkdir -p ~/.local/bin
cp dgx ~/.local/bin/
# Add to PATH if needed:
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# Option 3: System-wide install
go build -o dgx ./cmd/dgx
sudo cp dgx /usr/local/bin/
```

### Update Existing Installation

```bash
cd dgx-spark-cli
./update.sh
```

## Quick Start

### 1. Configure Your DGX Connection

```bash
dgx config set
```

The interactive setup will guide you through:
- **Hostname/IP** of your DGX Spark
- **SSH port** (default: 22)
- **Username** for SSH access
- **SSH key** - automatically detects existing keys or provides setup instructions

**Note**: When NVIDIA Sync is installed (macOS, Ubuntu, or Windows), `dgx config set` pre-loads the host, user, port, and Sync-managed SSH key (e.g., `~/Library/Application Support/NVIDIA/Sync/config/ssh_config` on macOS, `~/.local/share/NVIDIA/Sync/config/ssh_config` on Ubuntu, `%APPDATA%/NVIDIA/Sync/config/ssh_config` on Windows). On Arch—or any system without Sync—the wizard falls back to your standard `~/.ssh/id_ed25519` / `id_rsa` keys and shows you how to generate and upload a key if needed.

### 2. Test Connection

```bash
dgx status
```

### 3. Connect to DGX

```bash
dgx connect
```

## Usage

### Connection Management

```bash
# Open interactive SSH shell
dgx connect
# or
dgx ssh

# Check connection status
dgx status

# Show current configuration
dgx config show
```

### SSH Tunnel Management

```bash
# Create a tunnel: local port 8888 -> remote port 8888
dgx tunnel create 8888:8888 "Jupyter Notebook"

# List active tunnels
dgx tunnel list

# Kill a specific tunnel
dgx tunnel kill <PID>

# Kill all tunnels
dgx tunnel kill-all
```

#### Common Tunnel Examples

```bash
# Jupyter Notebook
dgx tunnel create 8888:8888 "Jupyter"

# TensorBoard
dgx tunnel create 6006:6006 "TensorBoard"

# VS Code Server
dgx tunnel create 8080:8080 "VSCode"

# JupyterLab
dgx tunnel create 8889:8889 "JupyterLab"
```

### GPU Monitoring

```bash
# Show formatted GPU status
dgx gpu

# Show raw nvidia-smi output
dgx gpu --raw

# Sample output:
# ┌─────────────────────────────────────────────────────────────────────┐
# │                         DGX GPU Status                              │
# ├─────────────────────────────────────────────────────────────────────┤
# │ GPU 0: NVIDIA GB100                                                 │
# │   Memory: 2048 MiB / 81920 MiB (Util: 15%)     Temp: 45°C          │
# │   Processes:                                                        │
# │     - PID 12345  python train.py              1024 MiB             │
# ├─────────────────────────────────────────────────────────────────────┤
# └─────────────────────────────────────────────────────────────────────┘
```

### File Synchronization

```bash
# Upload files to DGX
dgx sync ./local/path dgx:~/remote/path

# Download files from DGX
dgx sync dgx:~/remote/path ./local/path

# Sync with delete (removes extraneous files)
dgx sync --delete ./local/path dgx:~/remote/path
```

### DGX Spark Playbooks

Run AI/ML workloads with integrated playbook support:

```bash
# List available playbooks
dgx playbook list

# Ollama - Local model runner
dgx run ollama install
dgx run ollama pull qwen2.5:32b
dgx run ollama serve

# vLLM - Optimized inference
dgx run vllm pull
dgx run vllm serve meta-llama/Llama-2-7b-hf

# NVFP4 - 4-bit quantization
dgx run nvfp4 setup
dgx run nvfp4 quantize meta-llama/Llama-2-7b-hf

# Execute custom commands
dgx exec docker ps
dgx exec nvidia-smi
```

**See [PLAYBOOKS.md](PLAYBOOKS.md) for complete documentation and examples.**

## Workflow Examples

### Start a Jupyter Session

```bash
# 1. Create tunnel for Jupyter
dgx tunnel create 8888:8888 "Jupyter"

# 2. Connect and start Jupyter on DGX
dgx connect
# On DGX:
jupyter lab --no-browser --port 8888

# 3. Open browser to http://localhost:8888
```

### Monitor Training Jobs

```bash
# Check GPU status
dgx gpu

# Create TensorBoard tunnel if needed
dgx tunnel create 6006:6006 "TensorBoard"

# Upload training code
dgx sync ./my-model dgx:~/experiments/

# Connect and start training
dgx connect
```

### Development Workflow

```bash
# Create tunnels for common services
dgx tunnel create 8888:8888 "Jupyter"
dgx tunnel create 6006:6006 "TensorBoard"

# Sync code to DGX
dgx sync ./project dgx:~/work/project

# Monitor GPU usage
dgx gpu

# When done, clean up tunnels
dgx tunnel kill-all
```

## Configuration

Configuration is stored in `~/.config/dgx/config.yaml`:

```yaml
host: dgx-spark.example.com
port: 22
user: username
identity_file: /home/user/.ssh/id_ed25519
tunnels: []
```

You can edit this file manually or use `dgx config set`. If NVIDIA Sync metadata is present (macOS/Ubuntu/Windows), the CLI seeds this file automatically the first time you run it so those platforms work without additional prompts while other distros continue to use the standard SSH key locations.

## Development

### Project Structure

```
dgx-manager/
├── cmd/dgx/           # Main application entry point
├── internal/
│   ├── config/        # Configuration management
│   ├── ssh/           # SSH client implementation
│   ├── tunnel/        # Tunnel management
│   └── gpu/           # GPU monitoring
├── pkg/types/         # Shared types
├── Taskfile.yaml      # Build automation
└── README.md
```

### Build Commands

```bash
# Build
task build

# Run tests
task test

# Lint code
task lint

# Format code
task fmt

# Install locally
task install

# Build release binaries
task release
```

## Troubleshooting

### Connection Fails

```bash
# Verify SSH key permissions
chmod 600 ~/.ssh/id_ed25519

# Test SSH connection manually
ssh -i ~/.ssh/id_ed25519 user@dgx-host

# Check configuration
dgx config show
```

### Tunnel Port Already in Use

```bash
# List active tunnels
dgx tunnel list

# Kill specific tunnel
dgx tunnel kill <PID>

# Or use a different local port
dgx tunnel create 8889:8888 "Jupyter Alt Port"
```

### GPU Command Fails

```bash
# Ensure nvidia-smi is available on DGX
dgx connect
nvidia-smi  # Should work

# Try raw output for more details
dgx gpu --raw
```

## Tips & Tricks

### Shell Aliases

Add to your `~/.bashrc` or `~/.zshrc`:

```bash
alias dgx-gpu='dgx gpu'
alias dgx-ssh='dgx connect'
alias dgx-jupyter='dgx tunnel create 8888:8888 "Jupyter"'
alias dgx-tensorboard='dgx tunnel create 6006:6006 "TensorBoard"'
```

### Persistent Tunnels

Create a script to set up your common tunnels:

```bash
#!/bin/bash
# ~/bin/dgx-setup-tunnels.sh

dgx tunnel create 8888:8888 "Jupyter"
dgx tunnel create 6006:6006 "TensorBoard"
dgx tunnel create 8080:8080 "VSCode"
echo "All tunnels created"
dgx tunnel list
```

### Quick Status Check

```bash
# One-liner to check everything
dgx status && dgx gpu && dgx tunnel list
```

## License

MIT

## Contributing

Contributions welcome! This is a personal development tool but feel free to fork and customize for your needs.
