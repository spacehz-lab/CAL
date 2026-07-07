package cald

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/cald/daemon"
	"github.com/spacehz-lab/cal/internal/logging"
)

const (
	commandName = "cald"
	envHome     = "CAL_HOME"
)

const (
	flagHome = "home"
)

// CommandOptions configures the cald executable command tree.
type CommandOptions struct {
	Home    string
	Stdout  io.Writer
	Stderr  io.Writer
	Environ []string
}

// NewCommand builds the cald executable command tree.
func NewCommand(opts CommandOptions) (*cobra.Command, error) {
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

	cmd := &cobra.Command{
		Use:           commandName,
		Short:         "CAL local daemon",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.PersistentFlags().StringVar(&home, flagHome, home, "CAL home directory")
	cmd.AddCommand(newServeCommand(&home, stderr, env))
	return cmd, nil
}

func newServeCommand(home *string, stderr io.Writer, env []string) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the local CAL daemon",
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolvedHome, err := resolveHome(*home, env)
			if err != nil {
				return err
			}
			localDaemon, err := daemon.New(daemon.Options{
				Home:    resolvedHome,
				Logging: logging.Options{Name: commandName, Err: stderr, Env: env},
			})
			if err != nil {
				return err
			}
			return localDaemon.Serve(cmd.Context())
		},
	}
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
