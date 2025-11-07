package gpu

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/weatherman/dgx-manager/internal/ssh"
	"github.com/weatherman/dgx-manager/pkg/types"
)

// Monitor handles GPU status monitoring
type Monitor struct {
	sshClient *ssh.Client
}

// NewMonitor creates a new GPU monitor
func NewMonitor(sshClient *ssh.Client) *Monitor {
	return &Monitor{
		sshClient: sshClient,
	}
}

// GetStatus retrieves GPU status information
func (m *Monitor) GetStatus() ([]types.GPUInfo, error) {
	// Run nvidia-smi command
	output, err := m.sshClient.Execute("nvidia-smi --query-gpu=index,name,memory.used,memory.total,utilization.gpu,temperature.gpu --format=csv,noheader,nounits")
	if err != nil {
		return nil, fmt.Errorf("failed to query GPU: %w", err)
	}

	gpus, err := m.parseNvidiaSMI(output)
	if err != nil {
		return nil, err
	}

	// Get processes for each GPU
	for i := range gpus {
		processes, err := m.getGPUProcesses(gpus[i].ID)
		if err != nil {
			fmt.Printf("Warning: Failed to get processes for GPU %d: %v\n", gpus[i].ID, err)
		} else {
			gpus[i].Processes = processes
		}
	}

	return gpus, nil
}

// GetStatusText retrieves formatted GPU status as plain text
func (m *Monitor) GetStatusText() (string, error) {
	output, err := m.sshClient.Execute("nvidia-smi")
	if err != nil {
		return "", fmt.Errorf("failed to get GPU status: %w", err)
	}
	return output, nil
}

// parseNvidiaSMI parses nvidia-smi CSV output
func (m *Monitor) parseNvidiaSMI(output string) ([]types.GPUInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	gpus := make([]types.GPUInfo, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 6 {
			continue
		}

		id, err := strconv.Atoi(strings.TrimSpace(fields[0]))
		if err != nil {
			continue
		}

		gpu := types.GPUInfo{
			ID:          id,
			Name:        strings.TrimSpace(fields[1]),
			MemoryUsed:  strings.TrimSpace(fields[2]) + " MiB",
			MemoryTotal: strings.TrimSpace(fields[3]) + " MiB",
			Utilization: strings.TrimSpace(fields[4]) + "%",
			Temperature: strings.TrimSpace(fields[5]) + "°C",
		}

		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// getGPUProcesses retrieves processes running on a specific GPU
func (m *Monitor) getGPUProcesses(gpuID int) ([]types.GPUProcess, error) {
	cmd := fmt.Sprintf("nvidia-smi --query-compute-apps=pid,process_name,used_memory --format=csv,noheader,nounits --id=%d", gpuID)
	output, err := m.sshClient.Execute(cmd)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	processes := make([]types.GPUProcess, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 3 {
			continue
		}

		pid, err := strconv.Atoi(strings.TrimSpace(fields[0]))
		if err != nil {
			continue
		}

		process := types.GPUProcess{
			PID:         pid,
			Name:        strings.TrimSpace(fields[1]),
			MemoryUsage: strings.TrimSpace(fields[2]) + " MiB",
		}

		processes = append(processes, process)
	}

	return processes, nil
}

// GetGPUCount returns the number of GPUs
func (m *Monitor) GetGPUCount() (int, error) {
	output, err := m.sshClient.Execute("nvidia-smi --query-gpu=count --format=csv,noheader")
	if err != nil {
		return 0, err
	}

	count, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		// If that fails, try counting the number of GPUs
		gpus, err := m.GetStatus()
		if err != nil {
			return 0, err
		}
		return len(gpus), nil
	}

	return count, nil
}

// WatchGPU monitors GPU usage in real-time
func (m *Monitor) WatchGPU(interval int) error {
	// Run nvidia-smi in watch mode (dmon for device monitoring)
	cmd := fmt.Sprintf("watch -n %d nvidia-smi", interval)
	output, err := m.sshClient.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to watch GPU: %w", err)
	}

	fmt.Println(output)
	return nil
}

// FormatGPUStatus formats GPU information for display
func FormatGPUStatus(gpus []types.GPUInfo) string {
	var sb strings.Builder

	sb.WriteString("┌─────────────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│                         DGX GPU Status                              │\n")
	sb.WriteString("├─────────────────────────────────────────────────────────────────────┤\n")

	for _, gpu := range gpus {
		sb.WriteString(fmt.Sprintf("│ GPU %d: %-55s │\n", gpu.ID, gpu.Name))
		sb.WriteString(fmt.Sprintf("│   Memory: %s / %s (Util: %s)     Temp: %s       │\n",
			gpu.MemoryUsed, gpu.MemoryTotal, gpu.Utilization, gpu.Temperature))

		if len(gpu.Processes) > 0 {
			sb.WriteString("│   Processes:                                                       │\n")
			for _, proc := range gpu.Processes {
				procName := proc.Name
				if len(procName) > 30 {
					procName = procName[:27] + "..."
				}
				sb.WriteString(fmt.Sprintf("│     - PID %-6d %-30s %10s │\n",
					proc.PID, procName, proc.MemoryUsage))
			}
		}
		sb.WriteString("├─────────────────────────────────────────────────────────────────────┤\n")
	}

	sb.WriteString("└─────────────────────────────────────────────────────────────────────┘\n")

	return sb.String()
}

// ParseMemoryMiB extracts memory value in MiB from string like "1024 MiB"
func ParseMemoryMiB(memStr string) int {
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindStringSubmatch(memStr)
	if len(matches) > 1 {
		val, _ := strconv.Atoi(matches[1])
		return val
	}
	return 0
}
