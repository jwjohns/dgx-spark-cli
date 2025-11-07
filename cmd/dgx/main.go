package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/weatherman/dgx-manager/internal/config"
	"github.com/weatherman/dgx-manager/internal/gpu"
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
		// Check if config is required for this command
		requiresConfig := cmd.Name() != "config" && cmd.Name() != "version" && cmd.Name() != "help"
		if requiresConfig && !cfgManager.IsConfigured() {
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

		fmt.Println("Configure DGX Spark Connection")
		fmt.Println("================================")

		fmt.Print("Hostname/IP: ")
		fmt.Scanln(&cfg.Host)

		fmt.Print("Port [22]: ")
		var portStr string
		fmt.Scanln(&portStr)
		if portStr != "" {
			port, err := strconv.Atoi(portStr)
			if err == nil {
				cfg.Port = port
			}
		}

		fmt.Print("Username: ")
		fmt.Scanln(&cfg.User)

		fmt.Printf("SSH Key Path [%s]: ", cfg.IdentityFile)
		var keyPath string
		fmt.Scanln(&keyPath)
		if keyPath != "" {
			cfg.IdentityFile = keyPath
		}

		if err := cfgManager.Set(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to save config: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n✓ Configuration saved!")
		fmt.Printf("Config file: %s\n", cfgManager.GetConfigPath())
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
	Use:   "connect",
	Short: "Open an interactive SSH shell to DGX",
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
			fmt.Printf("✗ Connection failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ Connected (latency: %v)\n", latency)

		// Check for active tunnels
		tm := tunnel.NewManager(cfg)
		tunnels, _ := tm.List()
		fmt.Printf("Active tunnels: %d\n", len(tunnels))
	},
}

// tunnel command
var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Manage SSH tunnels",
	Aliases: []string{"t", "forward"},
}

var tunnelCreateCmd = &cobra.Command{
	Use:   "create <local-port>:<remote-port> [description]",
	Short: "Create a new SSH tunnel",
	Aliases: []string{"add", "new"},
	Args:  cobra.MinimumNArgs(1),
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
	Use:   "list",
	Short: "List active SSH tunnels",
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
	Use:   "kill <pid>",
	Short: "Kill a specific tunnel by PID",
	Aliases: []string{"stop", "rm"},
	Args:  cobra.ExactArgs(1),
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
		fmt.Println("✓ All tunnels terminated")
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

		fmt.Println("✓ Sync complete")
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

	// gpu flags
	gpuCmd.Flags().BoolP("raw", "r", false, "Show raw nvidia-smi output")

	// sync flags
	syncCmd.Flags().BoolP("delete", "d", false, "Delete extraneous files from destination")

	// Add all commands to root
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(tunnelCmd)
	rootCmd.AddCommand(gpuCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(versionCmd)
}
