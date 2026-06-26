package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
)

// DefaultStrategy is the stable v1 binding resolution strategy.
const DefaultStrategy = "default"

// Runner executes promoted or candidate bindings through runtime components.
type Runner struct {
	registry Registry
}

// ResolveOptions configures binding resolution.
type ResolveOptions struct {
	BindingID  string
	ProviderID string
	Strategy   string
}

// Resolution describes the selected binding and why it was selected.
type Resolution struct {
	Strategy           string       `json:"strategy"`
	BindingsConsidered int          `json:"bindings_considered"`
	Reason             string       `json:"reason"`
	Binding            core.Binding `json:"-"`
}

// NewRunner builds a runtime runner.
func NewRunner(registry Registry) Runner {
	return Runner{registry: registry}
}

// Verify checks one verifier through the runner registry.
func (runner Runner) Verify(ctx context.Context, verifier core.Verifier, inputs map[string]any) ([]core.EvidenceRef, map[string]any, error) {
	return runner.registry.Verify(ctx, verifier, inputs)
}

// Resolve chooses a promoted binding for one capability.
func (runner Runner) Resolve(capability core.Capability, opts ResolveOptions) (Resolution, error) {
	started := time.Now()
	strategy := opts.Strategy
	if strings.TrimSpace(strategy) == "" {
		strategy = DefaultStrategy
	}
	if strategy != DefaultStrategy {
		runner.logResolveFailed(capability, opts, strategy, "strategy", 0, started)
		return Resolution{}, fmt.Errorf("strategy %q is not supported", strategy)
	}

	candidates := make([]core.Binding, 0, len(capability.Bindings))
	for _, binding := range capability.Bindings {
		if binding.State != core.BindingStatePromoted {
			continue
		}
		if binding.Verifier == nil {
			continue
		}
		if !runner.Supports(binding.Execution.Kind) {
			continue
		}
		if opts.BindingID != "" && binding.ID != opts.BindingID {
			continue
		}
		if opts.ProviderID != "" && binding.ProviderID != opts.ProviderID {
			continue
		}
		candidates = append(candidates, binding)
	}
	if len(candidates) == 0 {
		runner.logResolveFailed(capability, opts, strategy, "no_binding", 0, started)
		return Resolution{
			Strategy:           strategy,
			BindingsConsidered: 0,
			Reason:             "no promoted binding with verifier and supported execution matched the request",
		}, fmt.Errorf("no promoted binding with verifier and supported execution matched capability %q", capability.ID)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ID < candidates[j].ID
	})
	runner.logResolveCompleted(capability, opts, strategy, candidates[0], len(candidates), started)
	return Resolution{
		Strategy:           strategy,
		BindingsConsidered: len(candidates),
		Reason:             "selected promoted binding with verifier and supported execution by deterministic binding id",
		Binding:            candidates[0],
	}, nil
}

func (runner Runner) logResolveCompleted(capability core.Capability, opts ResolveOptions, strategy string, binding core.Binding, considered int, started time.Time) {
	slog.Info("runtime binding resolved",
		"capability_id", capability.ID,
		"binding_filter", opts.BindingID,
		"provider_filter", opts.ProviderID,
		"strategy", strategy,
		"binding_id", binding.ID,
		"provider_id", binding.ProviderID,
		"bindings_considered", considered,
		"duration_ms", time.Since(started).Milliseconds(),
	)
}

func (runner Runner) logResolveFailed(capability core.Capability, opts ResolveOptions, strategy, stage string, considered int, started time.Time) {
	slog.Warn("runtime binding resolve failed",
		"capability_id", capability.ID,
		"binding_filter", opts.BindingID,
		"provider_filter", opts.ProviderID,
		"strategy", strategy,
		"bindings_considered", considered,
		"stage", stage,
		"duration_ms", time.Since(started).Milliseconds(),
	)
}
