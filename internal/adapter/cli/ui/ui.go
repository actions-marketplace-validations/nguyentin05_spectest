package ui

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/nguyentin05/spectest/internal/adapter/cli/cmdutil"
	adapterui "github.com/nguyentin05/spectest/internal/adapter/ui"
)

type uiOptions struct {
	host   string
	port   int
	noOpen bool
}

// NewCommand creates the `spectest ui` command to launch the web dashboard.
func NewCommand(f *cmdutil.Factory) *cobra.Command {
	opts := uiOptions{}

	cmd := &cobra.Command{
		Use:     "ui",
		Aliases: []string{"dashboard", "web"},
		Short:   "Launch the SpecTest interactive Web UI dashboard",
		Long: `Launches an interactive, local Web UI dashboard for SpecTest.

The UI provides full visual control over:
  - LLM credentials and provider settings
  - Target server healthcheck probing and Docker Compose management
  - OpenAPI, PRD, and test plan inspection
  - Real-time test suite execution and live progress streaming
  - Test cancellation, assertion inspection, and markdown matrix reporting`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			// Ensure port is available or find open port
			addr := fmt.Sprintf("%s:%d", opts.host, opts.port)
			l, err := net.Listen("tcp", addr)
			if err != nil {
				// Try auto-selecting an open port
				l, err = net.Listen("tcp", fmt.Sprintf("%s:0", opts.host))
				if err != nil {
					return fmt.Errorf("bind address on host %s: %w", opts.host, err)
				}
				opts.port = l.Addr().(*net.TCPAddr).Port
				addr = fmt.Sprintf("%s:%d", opts.host, opts.port)
			}
			_ = l.Close() // Close probe listener so server can bind

			serverURL := fmt.Sprintf("http://%s:%d", opts.host, opts.port)
			srv := adapterui.NewServer(addr, workDir, f)

			errCh := make(chan error, 1)
			go func() {
				if err := srv.Start(); err != nil {
					errCh <- err
				}
			}()

			_, _ = fmt.Fprintf(out, "SpecTest Web UI running at: %s\n", serverURL)
			_, _ = fmt.Fprintln(out, "Press Ctrl+C to stop the UI server.")

			if !opts.noOpen {
				_ = openBrowser(serverURL)
			}

			// Wait for interrupt signal
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

			select {
			case <-sigCh:
				_, _ = fmt.Fprintln(out, "\nGracefully shutting down Web UI server...")
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				return srv.Shutdown(ctx)
			case err := <-errCh:
				return fmt.Errorf("web server error: %w", err)
			}
		},
	}

	cmd.Flags().StringVar(&opts.host, "host", "127.0.0.1", "Host address to bind the web server")
	cmd.Flags().IntVarP(&opts.port, "port", "p", 8088, "Port number to listen on")
	cmd.Flags().BoolVar(&opts.noOpen, "no-open", false, "Do not automatically launch web browser")

	return cmd
}

// openBrowser launches the system default browser.
func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}
