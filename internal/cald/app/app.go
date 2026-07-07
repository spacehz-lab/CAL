package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/acquisition"
	"github.com/spacehz-lab/cal/internal/check"
	"github.com/spacehz-lab/cal/internal/config"
	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/entry"
	"github.com/spacehz-lab/cal/internal/eval"
	"github.com/spacehz-lab/cal/internal/execute"
	executecli "github.com/spacehz-lab/cal/internal/execute/cli"
	"github.com/spacehz-lab/cal/internal/llm"
	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/observe"
	observecli "github.com/spacehz-lab/cal/internal/observe/cli"
	"github.com/spacehz-lab/cal/internal/probe"
	"github.com/spacehz-lab/cal/internal/promote"
	"github.com/spacehz-lab/cal/internal/proposal"
	"github.com/spacehz-lab/cal/internal/proposal/replay"
	"github.com/spacehz-lab/cal/internal/proposal/rules"
	runpkg "github.com/spacehz-lab/cal/internal/run"
	"github.com/spacehz-lab/cal/internal/store"
	"github.com/spacehz-lab/cal/internal/tracelog"
	usepkg "github.com/spacehz-lab/cal/internal/use"
)

const (
	defaultWorkDir  = "work"
	apiKeyEnvPrefix = "env:"
)

var (
	ErrHomeRequired         = errors.New("cal home is required")
	ErrInvalidMode          = errors.New("invalid acquisition mode")
	ErrLLMNotConfigured     = errors.New("llm is not configured")
	ErrProposalPathRequired = errors.New("proposal path is required")
)

// App owns local cald application wiring and contract-facing methods.
type App struct {
	home     string
	workRoot string

	store    *store.Store
	registry *entry.Registry

	acquire *acquisitionRuntime
	llm     llm.Client
	run     *runpkg.Runner
	use     *usepkg.Runner
	eval    *eval.Runner
}

// Options configures one local cald application instance.
type Options struct {
	Home     string
	WorkRoot string
	LLM      *llm.Options
	Now      func() time.Time
}

// New builds a local cald application from concrete Release V1 packages.
func New(opts Options) (*App, error) {
	home := strings.TrimSpace(opts.Home)
	if home == "" {
		return nil, ErrHomeRequired
	}
	root, err := store.New(home)
	if err != nil {
		return nil, err
	}
	if err := root.Ensure(); err != nil {
		return nil, err
	}
	cfg, err := config.NewFile(home).Load()
	if err != nil {
		return nil, err
	}

	client, err := buildLLMClient(opts.LLM, cfg)
	if err != nil {
		return nil, err
	}

	checker := check.NewChecker()
	executor := execute.NewRunner(executecli.NewRunner())
	registry := entry.NewRegistry(root)
	observer := observe.NewRunner(map[model.ProviderKind]observe.Observer{
		model.ProviderKindCLI: observecli.NewObserver(),
	})
	prober := probe.NewRunner(executor, checker, probe.Options{})
	promoter := promote.NewRunner(root, opts.Now)
	tracer := tracelog.NewWriter(root, opts.Now)
	runner := runpkg.NewDefaultRunner(root, executor, checker, runpkg.WithProgress(logProgress))
	user := usepkg.NewDefaultRunner(root, runner, client, usepkg.WithProgress(logProgress))

	app := &App{
		home:     home,
		workRoot: workRoot(home, opts.WorkRoot),
		store:    root,
		registry: registry,
		acquire:  newAcquisitionRuntime(registry, root, observer, prober, promoter, tracer, logProgress),
		llm:      client,
		run:      runner,
		use:      user,
		eval:     eval.NewRunner(root),
	}
	return app, nil
}

// Home returns the resolved CAL home path used by this app.
func (app *App) Home() string {
	if app == nil {
		return ""
	}
	return app.home
}

// WorkRoot returns the acquisition work root used by this app.
func (app *App) WorkRoot() string {
	if app == nil {
		return ""
	}
	return app.workRoot
}

func buildLLMClient(runtime *llm.Options, cfg *config.Config) (llm.Client, error) {
	options, ok, err := llmOptions(runtime, cfg)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	client, err := llm.New(options)
	if err != nil {
		return nil, nil
	}
	return client, nil
}

func liveProposer(client llm.Client) *proposal.Runner {
	return proposal.NewLiveRunner(client, proposal.Options{})
}

type acquisitionRuntime struct {
	loader   acquisition.ProviderLoader
	catalog  acquisition.CatalogStore
	observer acquisition.Observer
	prober   acquisition.Prober
	promoter acquisition.Promoter
	tracer   acquisition.TraceWriter

	onProgress acquisition.ProgressFunc
}

func newAcquisitionRuntime(loader acquisition.ProviderLoader, catalog acquisition.CatalogStore, observer acquisition.Observer, prober acquisition.Prober, promoter acquisition.Promoter, tracer acquisition.TraceWriter, onProgress acquisition.ProgressFunc) *acquisitionRuntime {
	return &acquisitionRuntime{
		loader:     loader,
		catalog:    catalog,
		observer:   observer,
		prober:     prober,
		promoter:   promoter,
		tracer:     tracer,
		onProgress: onProgress,
	}
}

func (runtime *acquisitionRuntime) runner(proposer acquisition.Proposer) *acquisition.Runner {
	if runtime == nil || proposer == nil {
		return nil
	}
	return acquisition.NewRunner(runtime.loader, runtime.catalog, runtime.observer, proposer, runtime.prober, runtime.promoter, runtime.tracer, acquisition.WithProgress(runtime.onProgress))
}

func (app *App) proposerFor(req *contract.AcquisitionRequest) (acquisition.Proposer, error) {
	mode := acquisitionMode(req.Mode)
	switch mode {
	case contract.AcquisitionModeLive:
		if app.llm == nil {
			return nil, ErrLLMNotConfigured
		}
		return liveProposer(app.llm), nil
	case contract.AcquisitionModeReplay:
		if strings.TrimSpace(req.ProposalPath) == "" {
			return nil, ErrProposalPathRequired
		}
		return replay.NewRunner(req.ProposalPath), nil
	case contract.AcquisitionModeRules:
		return rules.NewRunner(), nil
	default:
		return nil, fmt.Errorf("%w %q", ErrInvalidMode, req.Mode)
	}
}

func workRoot(home string, configured string) string {
	if strings.TrimSpace(configured) != "" {
		return filepath.Clean(configured)
	}
	return filepath.Join(home, defaultWorkDir)
}
