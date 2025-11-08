package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/weatherman/dgx-manager/internal/config"
	"github.com/weatherman/dgx-manager/internal/gpu"
	"github.com/weatherman/dgx-manager/internal/playbook"
	"github.com/weatherman/dgx-manager/internal/ssh"
	"github.com/weatherman/dgx-manager/internal/tunnel"
	"github.com/weatherman/dgx-manager/pkg/types"
)

var (
	cfgManager *config.Manager
	Version    = "0.1.0"
)

func main() {
	var err error
	cfgManager, err = config.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize config: %v\n", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "dgx",
	Short: "DGX Spark management CLI",
	Long:  `A CLI tool to manage connections, tunnels, and GPU monitoring for DGX Spark.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Check if this command or its parent is one that doesn't require config
		cmdPath := cmd.CommandPath()
		noConfigRequired := strings.Contains(cmdPath, "config") ||
			strings.Contains(cmdPath, "version") ||
			strings.Contains(cmdPath, "help") ||
			strings.Contains(cmdPath, "completion")

		if !noConfigRequired && !cfgManager.IsConfigured() {
			fmt.Fprintf(os.Stderr, "Error: DGX not configured. Run 'dgx config set' first.\n")
			os.Exit(1)
		}
	},
}

// config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage DGX configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set DGX configuration interactively",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := cfgManager.Get()
		home, _ := os.UserHomeDir()

		defaultKeys := []string{
			filepath.Join(home, ".ssh", "id_ed25519"),
			filepath.Join(home, ".ssh", "id_rsa"),
		}

		profile, profileErr := config.DetectNVSyncProfile()
		if profileErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: unable to inspect NVIDIA Sync config: %v\n", profileErr)
		}
		if profile != nil {
			if cfg.Host == "" {
				cfg.Host = profile.Host
			}
			if cfg.Port == 0 {
				cfg.Port = profile.Port
			}
			if cfg.User == "" {
				cfg.User = profile.User
			}
			fmt.Println("Detected NVIDIA Sync configuration.")
			fmt.Printf("Defaults pulled from %s\n", profile.ConfigPath)
			fmt.Printf("DGX user: %s@%s (port %d)\n", cfg.User, cfg.Host, cfg.Port)
			fmt.Printf("Detected key: %s\n", profile.IdentityFile)
			fmt.Println("Press Enter to accept each default or type a new value.")
			fmt.Println()
		}

		fmt.Println("Configure DGX Spark Connection")
		fmt.Println("================================")
		fmt.Println()

		// Hostname
		fmt.Print("Hostname/IP: ")
		var host string
		fmt.Scanln(&host)
		if host != "" {
			cfg.Host = host
		}

		// Port
		fmt.Print("Port [22]: ")
		var portStr string
		fmt.Scanln(&portStr)
		if portStr != "" {
			port, err := strconv.Atoi(portStr)
			if err == nil {
				cfg.Port = port
			}
		} else if cfg.Port == 0 {
			cfg.Port = 22
		}

		// Username
		fmt.Print("Username: ")
		var user string
		fmt.Scanln(&user)
		if user != "" {
			cfg.User = user
		}

		// SSH Key
		fmt.Println()
		fmt.Println("SSH Key Setup")
		fmt.Println("-------------")

		keyConfigured := false
		if profile != nil {
			fmt.Printf("Use NVIDIA Sync SSH key at %s? [Y/n]: ", profile.IdentityFile)
			var useNVSync string
			fmt.Scanln(&useNVSync)
			if useNVSync == "" || strings.ToLower(useNVSync) == "y" {
				cfg.IdentityFile = profile.IdentityFile
				keyConfigured = true
			}
		}

		if !keyConfigured {
			// Check if default key exists
			var foundKey string
			for _, key := range defaultKeys {
				if _, err := os.Stat(key); err == nil {
					foundKey = key
					break
				}
			}

			if foundKey != "" {
				fmt.Printf("Found SSH key: %s\n", foundKey)
				fmt.Print("Use this key? [Y/n]: ")
				var useKey string
				fmt.Scanln(&useKey)
				if useKey == "" || strings.ToLower(useKey) == "y" {
					cfg.IdentityFile = foundKey
				} else {
					fmt.Print("Enter SSH key path: ")
					var customKey string
					fmt.Scanln(&customKey)
					if customKey != "" {
						cfg.IdentityFile = customKey
					}
				}
			} else {
				fmt.Println("Warning: No SSH key found in ~/.ssh/")
				fmt.Println()
				fmt.Println("To generate a new SSH key, run:")
				fmt.Println("  ssh-keygen -t ed25519 -C \"your-email@example.com\"")
				fmt.Println()
				fmt.Println("Then copy it to your DGX:")
				fmt.Printf("  ssh-copy-id %s@%s\n", cfg.User, cfg.Host)
				fmt.Println()
				fmt.Print("Enter SSH key path (or press Enter to use default): ")
				var keyPath string
				fmt.Scanln(&keyPath)
				if keyPath != "" {
					cfg.IdentityFile = keyPath
				} else {
					cfg.IdentityFile = filepath.Join(home, ".ssh", "id_ed25519")
				}
			}
		}

		// Validate minimum config
		if cfg.Host == "" || cfg.User == "" {
			fmt.Fprintf(os.Stderr, "\nError: Hostname and Username are required\n")
			os.Exit(1)
		}

		if err := cfgManager.Set(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to save config: %v\n", err)
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println("Configuration saved!")
		fmt.Printf("Config file: %s\n", cfgManager.GetConfigPath())
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  dgx status    # Test connection")
		fmt.Println("  dgx connect   # SSH to DGX")
		fmt.Println("  dgx gpu       # Check GPU status")
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := cfgManager.Get()
		fmt.Println("DGX Configuration:")
		fmt.Printf("  Host:         %s\n", cfg.Host)
		fmt.Printf("  Port:         %d\n", cfg.Port)
		fmt.Printf("  User:         %s\n", cfg.User)
		fmt.Printf("  Identity File: %s\n", cfg.IdentityFile)
		fmt.Printf("  Config Path:  %s\n", cfgManager.GetConfigPath())
	},
}

// connect command
var connectCmd = &cobra.Command{
	Use:     "connect",
	Short:   "Open an interactive SSH shell to DGX",
	Aliases: []string{"ssh"},
	Run: func(cmd *cobra.Command, args []string) {
		client, err := ssh.NewClient(cfgManager.Get())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Connecting to %s@%s...\n", cfgManager.Get().User, cfgManager.Get().Host)
		if err := client.InteractiveShell(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

// status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check DGX connection status",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := cfgManager.Get()
		client, err := ssh.NewClient(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Checking connection to %s@%s:%d...\n", cfg.User, cfg.Host, cfg.Port)
		latency, err := client.CheckConnection()
		if err != nil {
			fmt.Printf("Connection failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Connected (latency: %v)\n", latency)

		// Check for active tunnels
		tm := tunnel.NewManager(cfg)
		tunnels, _ := tm.List()
		fmt.Printf("Active tunnels: %d\n", len(tunnels))
	},
}

// tunnel command
var tunnelCmd = &cobra.Command{
	Use:     "tunnel",
	Short:   "Manage SSH tunnels",
	Aliases: []string{"t", "forward"},
}

var tunnelCreateCmd = &cobra.Command{
	Use:     "create <local-port>:<remote-port> [description]",
	Short:   "Create a new SSH tunnel",
	Aliases: []string{"add", "new"},
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		parts := strings.Split(args[0], ":")
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Error: Invalid format. Use <local-port>:<remote-port>\n")
			os.Exit(1)
		}

		localPort, err := strconv.Atoi(parts[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid local port: %s\n", parts[0])
			os.Exit(1)
		}

		remotePort, err := strconv.Atoi(parts[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid remote port: %s\n", parts[1])
			os.Exit(1)
		}

		description := ""
		if len(args) > 1 {
			description = strings.Join(args[1:], " ")
		}

		tm := tunnel.NewManager(cfgManager.Get())

		// Check if port is already in use
		if tm.IsPortInUse(localPort) {
			fmt.Fprintf(os.Stderr, "Error: Local port %d is already in use\n", localPort)
			os.Exit(1)
		}

		t := types.Tunnel{
			ID:          fmt.Sprintf("tunnel-%d", time.Now().Unix()),
			LocalPort:   localPort,
			RemotePort:  remotePort,
			RemoteHost:  "localhost",
			Description: description,
		}

		if err := tm.Create(t); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Save to config
		cfgManager.AddTunnel(t)
	},
}

var tunnelListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List active SSH tunnels",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		tm := tunnel.NewManager(cfgManager.Get())
		tunnels, err := tm.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(tunnels) == 0 {
			fmt.Println("No active tunnels")
			return
		}

		fmt.Println("Active SSH Tunnels:")
		fmt.Println("-------------------")
		for _, t := range tunnels {
			fmt.Printf("PID %d: localhost:%d -> %s:%d\n",
				t.PID, t.LocalPort, t.RemoteHost, t.RemotePort)
		}
	},
}

var tunnelKillCmd = &cobra.Command{
	Use:     "kill <pid>",
	Short:   "Kill a specific tunnel by PID",
	Aliases: []string{"stop", "rm"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pid, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid PID: %s\n", args[0])
			os.Exit(1)
		}

		tm := tunnel.NewManager(cfgManager.Get())
		if err := tm.Kill(pid); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var tunnelKillAllCmd = &cobra.Command{
	Use:   "kill-all",
	Short: "Kill all active tunnels",
	Run: func(cmd *cobra.Command, args []string) {
		tm := tunnel.NewManager(cfgManager.Get())
		if err := tm.KillAll(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("All tunnels terminated")
	},
}

// gpu command
var gpuCmd = &cobra.Command{
	Use:   "gpu",
	Short: "Monitor GPU status",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := ssh.NewClient(cfgManager.Get())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer client.Close()

		monitor := gpu.NewMonitor(client)

		// Check if --raw flag is set
		raw, _ := cmd.Flags().GetBool("raw")

		if raw {
			output, err := monitor.GetStatusText()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(output)
		} else {
			gpus, err := monitor.GetStatus()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			fmt.Println(gpu.FormatGPUStatus(gpus))
		}
	},
}

// sync command
var syncCmd = &cobra.Command{
	Use:   "sync <source> <destination>",
	Short: "Sync files between local and DGX",
	Long: `Sync files using rsync.
Examples:
  dgx sync ./code dgx:~/projects/  # Upload to DGX
  dgx sync dgx:~/results ./        # Download from DGX`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := ssh.NewClient(cfgManager.Get())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		source := args[0]
		dest := args[1]
		cfg := cfgManager.Get()

		// Replace "dgx:" with actual SSH path
		source = strings.ReplaceAll(source, "dgx:", fmt.Sprintf("%s@%s:", cfg.User, cfg.Host))
		dest = strings.ReplaceAll(dest, "dgx:", fmt.Sprintf("%s@%s:", cfg.User, cfg.Host))

		deleteFlag, _ := cmd.Flags().GetBool("delete")

		fmt.Printf("Syncing %s -> %s\n", args[0], args[1])
		if err := client.Rsync(source, dest, deleteFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Sync complete")
	},
}

// setup-key command
var setupKeyCmd = &cobra.Command{
	Use:   "setup-key",
	Short: "Setup SSH key authentication with DGX",
	Long:  `Helps you copy your SSH public key to the DGX for passwordless authentication.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := cfgManager.Get()

		// Check if public key exists
		pubKeyPath := cfg.IdentityFile + ".pub"
		pubKeyData, err := os.ReadFile(pubKeyPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Cannot read public key at %s\n", pubKeyPath)
			fmt.Fprintf(os.Stderr, "Make sure your SSH key pair exists.\n")
			os.Exit(1)
		}

		fmt.Println("SSH Key Setup for DGX")
		fmt.Println("======================")
		fmt.Println()
		fmt.Printf("Your public key: %s\n", pubKeyPath)
		fmt.Println(string(pubKeyData))
		fmt.Println()
		fmt.Println("To enable passwordless SSH access, you need to copy this key to your DGX.")
		fmt.Println()
		fmt.Println("Option 1: Automatic (requires password)")
		fmt.Println("  Run this command and enter your DGX password when prompted:")
		fmt.Printf("  ssh-copy-id -i %s %s@%s\n", pubKeyPath, cfg.User, cfg.Host)
		fmt.Println()
		fmt.Println("Option 2: Manual")
		fmt.Println("  1. SSH to your DGX with password:")
		fmt.Printf("     ssh %s@%s\n", cfg.User, cfg.Host)
		fmt.Println("  2. On the DGX, run:")
		fmt.Println("     mkdir -p ~/.ssh && chmod 700 ~/.ssh")
		fmt.Println("     echo 'YOUR_PUBLIC_KEY' >> ~/.ssh/authorized_keys")
		fmt.Println("     chmod 600 ~/.ssh/authorized_keys")
		fmt.Println()
		fmt.Print("Would you like to try automatic setup now? [Y/n]: ")

		var response string
		fmt.Scanln(&response)

		if response == "" || strings.ToLower(response) == "y" {
			fmt.Println()
			fmt.Println("Attempting to copy SSH key...")
			fmt.Println("(You will be prompted for your DGX password)")
			fmt.Println()

			// Use ssh-copy-id with STDIN for password
			cmd := exec.Command("ssh-copy-id", "-i", pubKeyPath, fmt.Sprintf("%s@%s", cfg.User, cfg.Host))
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "\nWarning: Automatic setup failed.\n")
				fmt.Fprintf(os.Stderr, "Please use the manual method shown above.\n")
				os.Exit(1)
			}

			fmt.Println()
			fmt.Println("SSH key copied successfully!")
			fmt.Println()
			fmt.Println("Test your connection:")
			fmt.Println("  dgx status")
			fmt.Println("  dgx gpu")
		} else {
			fmt.Println()
			fmt.Println("Use one of the methods above to copy your SSH key manually.")
		}
	},
}

// playbook command
var playbookCmd = &cobra.Command{
	Use:     "playbook",
	Aliases: []string{"pb"},
	Short:   "Manage DGX Spark playbooks",
}

var playbookListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List available playbooks",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		playbooks := playbook.GetAvailablePlaybooks()

		// Group by category
		categories := make(map[string][]playbook.Playbook)
		for _, pb := range playbooks {
			categories[pb.Category] = append(categories[pb.Category], pb)
		}

		fmt.Println("Available DGX Spark Playbooks")
		fmt.Println("=============================")
		fmt.Println()

		for category, pbs := range categories {
			fmt.Printf("## %s\n", category)
			for _, pb := range pbs {
				fmt.Printf("  %-25s %s\n", pb.Name, pb.Description)
			}
			fmt.Println()
		}

		fmt.Println("Usage:")
		fmt.Println("  dgx run <playbook> <command> [args...]")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  dgx run ollama install")
		fmt.Println("  dgx run ollama pull qwen2.5:32b")
		fmt.Println("  dgx run vllm serve meta-llama/Llama-2-7b-hf")
		fmt.Println("  dgx run nvfp4 quantize meta-llama/Llama-2-7b-hf")
	},
}

// run command
var runCmd = &cobra.Command{
	Use:   "run <playbook> <command> [args...]",
	Short: "Run a DGX Spark playbook",
	Long: `Execute playbooks for various AI/ML workloads on your DGX Spark.

Available playbooks:
  ollama  - Local model runner (install, pull, serve, run)
  vllm    - Optimized LLM inference (pull, serve, status)
  nvfp4   - 4-bit quantization (setup, quantize)
  dmr     - Docker Model Runner (setup, install, pull, run, status, logs)

Examples:
  dgx run ollama install
  dgx run ollama pull qwen2.5:32b
  dgx run vllm serve meta-llama/Llama-2-7b-hf
  dgx run nvfp4 quantize meta-llama/Llama-2-7b-hf
  dgx run dmr status`,
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 || isHelpArg(args[0]) {
			cmd.Help()
			return
		}

		client, err := ssh.NewClient(cfgManager.Get())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer client.Close()

		manager := playbook.NewManager(client)
		playbookName := args[0]
		playbookArgs := args[1:]
		if len(playbookArgs) > 0 && isHelpArg(playbookArgs[0]) {
			playbook.PrintHelp(playbookName)
			return
		}

		if err := manager.Execute(playbookName, playbookArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help" || strings.EqualFold(arg, "help")
}

func promptForSecret(label string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter %s: ", label)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s cannot be empty", label)
	}
	return value, nil
}

func setRemoteEnvVar(varName, value string) error {
	client, err := ssh.NewClient(cfgManager.Get())
	if err != nil {
		return err
	}
	defer client.Close()

	encoded := base64.StdEncoding.EncodeToString([]byte(value))
	script := fmt.Sprintf(`
import base64, os, pathlib, shlex

name = "%s"
value = base64.b64decode(os.environ["ENV_VALUE"]).decode()

config_dir = pathlib.Path.home() / ".config" / "dgx"
env_file = config_dir / "env.sh"
config_dir.mkdir(parents=True, exist_ok=True)

lines = []
if env_file.exists():
    for line in env_file.read_text().splitlines():
        if not line.startswith(f"export {name}="):
            lines.append(line)
lines.append(f"export {name}={shlex.quote(value)}")
env_file.write_text("\n".join(lines) + "\n")

bashrc = pathlib.Path.home() / ".bashrc"
source_line = "source ~/.config/dgx/env.sh"
if bashrc.exists():
    content = bashrc.read_text()
else:
    content = ""
if source_line not in content:
    with bashrc.open("a") as fh:
        if content and not content.endswith("\n"):
            fh.write("\n")
        fh.write(source_line + "\n")

print(f"Stored {name} in {env_file} and ensured {bashrc} sources it.")
`, varName)

	command := fmt.Sprintf("ENV_VALUE=%s python3 - <<'PY'\n%s\nPY", shellQuote(encoded), script)
	output, err := client.Execute(command)
	if err != nil {
		return fmt.Errorf("remote update failed: %w", err)
	}
	if strings.TrimSpace(output) != "" {
		fmt.Println(strings.TrimSpace(output))
	}
	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

// env command
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment tokens on your DGX",
}

var envHFTokenCmd = &cobra.Command{
	Use:   "hf-token",
	Short: "Set HF_TOKEN on the DGX",
	Run: func(cmd *cobra.Command, args []string) {
		value, _ := cmd.Flags().GetString("value")
		if value == "" {
			var err error
			value, err = promptForSecret("Hugging Face token")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
		if err := setRemoteEnvVar("HF_TOKEN", value); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var envWandbCmd = &cobra.Command{
	Use:   "wandb",
	Short: "Set WANDB_API_KEY on the DGX",
	Run: func(cmd *cobra.Command, args []string) {
		value, _ := cmd.Flags().GetString("value")
		if value == "" {
			var err error
			value, err = promptForSecret("Weights & Biases API key")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
		if err := setRemoteEnvVar("WANDB_API_KEY", value); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

// exec command for running arbitrary commands
var execCmd = &cobra.Command{
	Use:   "exec <command>",
	Short: "Execute a command on the DGX",
	Long:  `Run an arbitrary shell command on your DGX Spark.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := ssh.NewClient(cfgManager.Get())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer client.Close()

		command := strings.Join(args, " ")
		output, err := client.Execute(command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(output)
	},
}

// version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("dgx version %s\n", Version)
	},
}

func init() {
	// config subcommands
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)

	// tunnel subcommands
	tunnelCmd.AddCommand(tunnelCreateCmd)
	tunnelCmd.AddCommand(tunnelListCmd)
	tunnelCmd.AddCommand(tunnelKillCmd)
	tunnelCmd.AddCommand(tunnelKillAllCmd)

	// playbook subcommands
	playbookCmd.AddCommand(playbookListCmd)

	// gpu flags
	gpuCmd.Flags().BoolP("raw", "r", false, "Show raw nvidia-smi output")

	// sync flags
	syncCmd.Flags().BoolP("delete", "d", false, "Delete extraneous files from destination")

	// env subcommands
	envHFTokenCmd.Flags().String("value", "", "Token to set (omit to be prompted)")
	envWandbCmd.Flags().String("value", "", "API key to set (omit to be prompted)")
	envCmd.AddCommand(envHFTokenCmd)
	envCmd.AddCommand(envWandbCmd)

	// Add all commands to root
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(tunnelCmd)
	rootCmd.AddCommand(gpuCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(setupKeyCmd)
	rootCmd.AddCommand(playbookCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(envCmd)
}
