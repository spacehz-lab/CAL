package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/cli/client"
	"github.com/spacehz-lab/cal/internal/logging"
)

const (
	commandName = "calctl"
	envHome     = "CAL_HOME"
)

const (
	flagHome   = "home"
	flagJSON   = "json"
	flagStream = "stream"
)

// Options configures one calctl command tree.
type Options struct {
	Home    string
	Stdout  io.Writer
	Stderr  io.Writer
	Environ []string
}

// CLI owns the user-facing calctl command surface.
type CLI struct {
	home   string
	stdout io.Writer
	stderr io.Writer
	env    []string
}

// New creates one CLI command owner.
func New(opts Options) (*CLI, error) {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	env := opts.Environ
	if env == nil {
		env = os.Environ()
	}
	home, err := resolveHome(opts.Home, env)
	if err != nil {
		return nil, err
	}
	cli := &CLI{home: home, stdout: stdout, stderr: stderr, env: env}
	logging.Configure(&logging.Options{Name: commandName, Err: stderr, Env: env})
	return cli, nil
}

// Command builds the calctl command tree.
func (cli *CLI) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:           commandName,
		Short:         "Capability Acquisition Loop",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(cli.stdout)
	cmd.SetErr(cli.stderr)
	cmd.PersistentFlags().StringVar(&cli.home, flagHome, cli.home, "CAL home directory")
	cmd.AddCommand(cli.newDaemonCommand())
	cmd.AddCommand(cli.newProvidersCommand())
	cmd.AddCommand(cli.newAcquisitionCommand())
	cmd.AddCommand(cli.newCapabilitiesCommand())
	cmd.AddCommand(cli.newRunsCommand())
	cmd.AddCommand(cli.newUseCommand())
	cmd.AddCommand(cli.newEvalCommand())
	return cmd
}

type commandContext struct {
	client *client.Client
}

func (cli *CLI) commandContext() (*commandContext, error) {
	home, err := resolveHome(cli.home, cli.env)
	if err != nil {
		return nil, err
	}
	daemonClient, err := client.New(client.Options{Home: home})
	if err != nil {
		return nil, err
	}
	return &commandContext{client: daemonClient}, nil
}

func resolveHome(explicit string, env []string) (string, error) {
	if home := strings.TrimSpace(explicit); home != "" {
		return filepath.Clean(home), nil
	}
	if home := strings.TrimSpace(envValue(env, envHome)); home != "" {
		return filepath.Clean(home), nil
	}
	return defaultHome(env)
}

func defaultHome(env []string) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "cal"), nil
	case "windows":
		if dir := strings.TrimSpace(envValue(env, "LocalAppData")); dir != "" {
			return filepath.Join(dir, "cal"), nil
		}
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, "cal"), nil
	default:
		if dir := strings.TrimSpace(envValue(env, "XDG_DATA_HOME")); dir != "" {
			return filepath.Join(dir, "cal"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", "cal"), nil
	}
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return item[len(prefix):]
		}
	}
	return ""
}

func required(value string, name string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}
