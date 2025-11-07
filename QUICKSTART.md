# DGX Manager - Quick Start Guide

## Installation

```bash
cd ~/Development/dgx-manager

# Option 1: Using the install script
./install.sh

# Option 2: Manual build and install
go build -o dgx ./cmd/dgx
sudo cp dgx /usr/local/bin/

# Option 3: Using Task (if installed)
task install
```

## Initial Setup

### 1. Configure DGX Connection

```bash
dgx config set
```

Enter your DGX Spark details:
- **Hostname/IP**: The IP address or hostname of your DGX Spark
- **Port**: SSH port (usually 22)
- **Username**: Your SSH username
- **SSH Key Path**: Path to your private key (default: `~/.ssh/id_ed25519`)

### 2. Test Connection

```bash
dgx status
```

Expected output:
```
Checking connection to user@dgx-spark:22...
✓ Connected (latency: 15ms)
Active tunnels: 0
```

## Common Workflows

### Jupyter Notebook Development

```bash
# 1. Create tunnel for Jupyter
dgx tunnel create 8888:8888 "Jupyter Notebook"

# 2. SSH to DGX and start Jupyter
dgx connect
# On DGX:
jupyter notebook --no-browser --port 8888 --ip 0.0.0.0

# 3. Open browser on Surface
# Navigate to: http://localhost:8888

# 4. Check GPU usage while training
dgx gpu

# 5. When done, kill tunnel
dgx tunnel list
dgx tunnel kill <PID>
```

### Model Training Workflow

```bash
# 1. Check GPU availability
dgx gpu

# 2. Sync training code to DGX
dgx sync ./my-model dgx:~/experiments/my-model

# 3. Set up monitoring tunnels
dgx tunnel create 6006:6006 "TensorBoard"

# 4. Connect and start training
dgx connect
# On DGX:
cd ~/experiments/my-model
python train.py

# 5. Monitor on Surface via TensorBoard
# Navigate to: http://localhost:6006

# 6. Download results
dgx sync dgx:~/experiments/my-model/results ./results
```

### Quick GPU Check

```bash
# Formatted output
dgx gpu

# Raw nvidia-smi output
dgx gpu --raw
```

### File Management

```bash
# Upload entire project
dgx sync ./project dgx:~/work/

# Download results directory
dgx sync dgx:~/work/results ./

# Sync with delete (mirror)
dgx sync --delete ./local dgx:~/remote
```

## Troubleshooting

### "DGX not configured" error

```bash
dgx config set
```

### SSH key permission errors

```bash
chmod 600 ~/.ssh/id_ed25519
```

### Port already in use

```bash
# List and kill existing tunnels
dgx tunnel list
dgx tunnel kill <PID>

# Or use different port
dgx tunnel create 8889:8888
```

### Connection timeout

```bash
# Check if SSH works manually
ssh -i ~/.ssh/id_ed25519 user@dgx-host

# Verify config
dgx config show
```

## Tips

### Shell Aliases

Add to `~/.bashrc`:

```bash
alias dgx-gpu='dgx gpu'
alias dgx-ssh='dgx connect'
alias dgx-j='dgx tunnel create 8888:8888 "Jupyter"'
alias dgx-tb='dgx tunnel create 6006:6006 "TensorBoard"'
```

### Check Everything

```bash
dgx status && dgx gpu && dgx tunnel list
```

### Persistent Tunnel Setup

Create `~/bin/dgx-tunnels.sh`:

```bash
#!/bin/bash
dgx tunnel create 8888:8888 "Jupyter"
dgx tunnel create 6006:6006 "TensorBoard"
dgx tunnel create 8080:8080 "VSCode Server"
dgx tunnel list
```

## Next Steps

- Read the full [README.md](README.md) for detailed documentation
- Check available commands: `dgx --help`
- Explore command options: `dgx tunnel --help`, `dgx gpu --help`, etc.
