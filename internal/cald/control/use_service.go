package control

import (
	"context"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	sharedllm "github.com/spacehz-lab/cal/internal/llm"
	caluse "github.com/spacehz-lab/cal/internal/use"
)

// Use routes an intent to one promoted capability binding, then delegates to Run.
func (svc Service) Use(ctx context.Context, req caluse.Request) (caluse.Result, error) {
	if reqErr := req.Validate(); reqErr != nil {
		return caluse.Result{}, NewAPIError(reqErr.Code, reqErr.Message)
	}

	started := time.Now().UTC()
	result := caluse.NewResult(req.Intent, started)

	capabilities, err := svc.store.ListCapabilities()
	if err != nil {
		return caluse.Result{}, err
	}
	resolver, err := svc.newUseResolver(req)
	if err != nil {
		return caluse.Result{}, err
	}
	resolution, selectErr := resolver.Resolve(ctx, capabilities)
	if selectErr != nil {
		result.Fail(started, selectErr.Code, selectErr.Message)
		return result, nil
	}
	result.Selection = &resolution.Selection

	inputs, planErr := caluse.NewPlanner(result.ID, started).Plan(req, resolution)
	if planErr != nil {
		result.Fail(started, planErr.Code, planErr.Message)
		return result, nil
	}

	run, err := svc.Run(ctx, RunRequest{
		CapabilityID: resolution.Selection.CapabilityID,
		BindingID:    resolution.Selection.BindingID,
		Inputs:       inputs,
		ProviderID:   req.ProviderID,
		Strategy:     req.Strategy,
		Verify:       req.Verify,
	})
	if err != nil {
		return caluse.Result{}, err
	}
	result.Run = &run
	if run.Status == core.RunStatusFailed {
		code := "run_failed"
		message := "selected binding run failed"
		if run.Error != nil {
			code = run.Error.Code
			message = run.Error.Message
		}
		result.Fail(started, code, message)
		return result, nil
	}
	result.Complete(started)
	return result, nil
}

func (svc Service) newUseResolver(req caluse.Request) (caluse.Resolver, error) {
	cfg, err := svc.cfg.Load()
	if err != nil {
		return caluse.Resolver{}, err
	}
	llmConfig, err := cfg.RuntimeLLMConfig()
	if err != nil {
		return caluse.Resolver{}, NewAPIError("invalid_llm_config", err.Error())
	}
	client, err := sharedllm.NewClient(llmConfig)
	if err != nil {
		return caluse.Resolver{}, NewAPIError("invalid_llm_config", err.Error())
	}
	if client == nil {
		return caluse.NewResolver(req), nil
	}
	return caluse.NewResolver(req, caluse.WithLLM(client)), nil
}
