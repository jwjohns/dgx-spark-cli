package ssh

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/weatherman/dgx-manager/pkg/types"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Client manages SSH connections to the DGX
type Client struct {
	config *types.Config
	client *ssh.Client
}

// NewClient creates a new SSH client
func NewClient(config *types.Config) (*Client, error) {
	return &Client{
		config: config,
	}, nil
}

// Connect establishes an SSH connection
func (c *Client) Connect() error {
	// Load SSH key
	key, err := os.ReadFile(c.config.IdentityFile)
	if err != nil {
		return fmt.Errorf("failed to read SSH key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return fmt.Errorf("failed to parse SSH key: %w", err)
	}

	// Load known_hosts
	home, _ := os.UserHomeDir()
	knownHostsPath := fmt.Sprintf("%s/.ssh/known_hosts", home)
	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		// If known_hosts doesn't exist, use insecure (warn user)
		fmt.Fprintf(os.Stderr, "Warning: Using insecure host key verification\n")
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	// SSH client configuration
	sshConfig := &ssh.ClientConfig{
		User: c.config.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	// Connect
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	c.client = client
	return nil
}

// Close closes the SSH connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Execute runs a command on the remote host
func (c *Client) Execute(command string) (string, error) {
	if c.client == nil {
		if err := c.Connect(); err != nil {
			return "", err
		}
		defer c.Close()
	}

	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

// InteractiveShell starts an interactive SSH shell
func (c *Client) InteractiveShell() error {
	// Use native SSH command for interactive shell (better terminal handling)
	args := []string{
		"-i", c.config.IdentityFile,
		"-p", fmt.Sprintf("%d", c.config.Port),
		fmt.Sprintf("%s@%s", c.config.User, c.config.Host),
	}

	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// CheckConnection tests the connection without keeping it open
func (c *Client) CheckConnection() (time.Duration, error) {
	start := time.Now()

	if err := c.Connect(); err != nil {
		return 0, err
	}
	defer c.Close()

	latency := time.Since(start)
	return latency, nil
}

// ForwardPort creates an SSH tunnel
func (c *Client) ForwardPort(localPort, remotePort int, remoteHost string) error {
	if c.client == nil {
		if err := c.Connect(); err != nil {
			return err
		}
	}

	// Listen on local port
	localAddr := fmt.Sprintf("localhost:%d", localPort)
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", localAddr, err)
	}

	go func() {
		defer listener.Close()
		for {
			localConn, err := listener.Accept()
			if err != nil {
				return
			}

			go c.handleForward(localConn, remoteHost, remotePort)
		}
	}()

	return nil
}

// handleForward handles a single forwarded connection
func (c *Client) handleForward(localConn net.Conn, remoteHost string, remotePort int) {
	defer localConn.Close()

	remoteAddr := fmt.Sprintf("%s:%d", remoteHost, remotePort)
	remoteConn, err := c.client.Dial("tcp", remoteAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to remote %s: %v\n", remoteAddr, err)
		return
	}
	defer remoteConn.Close()

	// Copy data bidirectionally
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(remoteConn, localConn)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(localConn, remoteConn)
		done <- struct{}{}
	}()

	<-done
}

// CopyFile transfers a file using SCP
func (c *Client) CopyFile(source, dest string) error {
	args := []string{
		"-i", c.config.IdentityFile,
		"-P", fmt.Sprintf("%d", c.config.Port),
		"-r",
		source,
		dest,
	}

	cmd := exec.Command("scp", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Rsync syncs files using rsync over SSH
func (c *Client) Rsync(source, dest string, deleteExtraneous bool) error {
	args := []string{
		"-avz",
		"--progress",
		"-e", fmt.Sprintf("ssh -i %s -p %d", c.config.IdentityFile, c.config.Port),
	}

	if deleteExtraneous {
		args = append(args, "--delete")
	}

	args = append(args, source, dest)

	cmd := exec.Command("rsync", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
