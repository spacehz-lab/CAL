package client

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/cald/endpoint"
	"github.com/spacehz-lab/cal/internal/contract"
)

const (
	defaultTimeout = 20 * time.Minute
)

const (
	routeDaemonStatus       = "/v1/daemon/status"
	routeDaemonStop         = "/v1/daemon/stop"
	routeProviders          = "/v1/providers"
	routeCapabilities       = "/v1/capabilities"
	routeAcquisitions       = "/v1/acquisitions"
	routeAcquisitionsStream = "/v1/acquisitions/stream"
	routeRuns               = "/v1/runs"
	routeRunsStream         = "/v1/runs/stream"
	routeUses               = "/v1/uses"
	routeUsesStream         = "/v1/uses/stream"
	routeEval               = "/v1/eval"
)

const (
	queryCapabilityID = "capability_id"
	queryProviderID   = "provider_id"
)

// Options configures one CLI-internal daemon client.
type Options struct {
	Home    string
	BaseURL string
	HTTP    *http.Client
	Timeout time.Duration
}

// Client calls the local cald HTTP API for CLI commands.
type Client struct {
	baseURL string
	http    *http.Client
}

// New creates a local daemon client from an explicit base URL or endpoint file.
func New(opts Options) (*Client, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/")
	if baseURL == "" {
		record, ok, err := endpoint.Read(opts.Home)
		if err != nil {
			return nil, newError(0, contract.ErrorCaldUnavailable, err.Error())
		}
		if !ok {
			return nil, newError(0, contract.ErrorCaldUnavailable, "cald is not running")
		}
		baseURL = strings.TrimRight(strings.TrimSpace(record.BaseURL), "/")
	}
	if baseURL == "" {
		return nil, newError(0, contract.ErrorCaldUnavailable, "cald endpoint is empty")
	}
	return &Client{baseURL: baseURL, http: httpClient(opts.HTTP, opts.Timeout)}, nil
}

// Status returns local daemon status.
func (client *Client) Status(ctx context.Context) (*contract.DaemonStatus, error) {
	var response contract.DaemonStatus
	if err := client.get(ctx, routeDaemonStatus, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Stop requests local daemon shutdown.
func (client *Client) Stop(ctx context.Context) (*contract.DaemonStopResponse, error) {
	var response contract.DaemonStopResponse
	if err := client.post(ctx, routeDaemonStop, map[string]any{}, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// AddProvider registers one explicit provider path.
func (client *Client) AddProvider(ctx context.Context, req *contract.AddProviderRequest) (*contract.ProviderListResponse, error) {
	var response contract.ProviderListResponse
	if err := client.post(ctx, routeProviders, req, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// ListProviders returns registered providers.
func (client *Client) ListProviders(ctx context.Context) (*contract.ProviderListResponse, error) {
	var response contract.ProviderListResponse
	if err := client.get(ctx, routeProviders, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// ListCapabilities returns promoted capability summaries.
func (client *Client) ListCapabilities(ctx context.Context, req *contract.CapabilityListRequest) (*contract.CapabilityListResponse, error) {
	var response contract.CapabilityListResponse
	if err := client.get(ctx, routeCapabilities, capabilityQuery(req), &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Acquire starts one acquisition request.
func (client *Client) Acquire(ctx context.Context, req *contract.AcquisitionRequest) (*contract.AcquisitionResponse, error) {
	var response contract.AcquisitionResponse
	if err := client.post(ctx, routeAcquisitions, req, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// AcquireStream starts one acquisition request and reports live progress events.
func (client *Client) AcquireStream(ctx context.Context, req *contract.AcquisitionRequest, onEvent StreamHandler) (*contract.AcquisitionResponse, error) {
	var response contract.AcquisitionResponse
	if err := client.stream(ctx, routeAcquisitionsStream, req, onEvent, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Run executes one promoted capability binding.
func (client *Client) Run(ctx context.Context, req *contract.RunRequest) (*contract.RunResponse, error) {
	var response contract.RunResponse
	if err := client.post(ctx, routeRuns, req, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// RunStream executes one promoted capability binding and reports live progress events.
func (client *Client) RunStream(ctx context.Context, req *contract.RunRequest, onEvent StreamHandler) (*contract.RunResponse, error) {
	var response contract.RunResponse
	if err := client.stream(ctx, routeRunsStream, req, onEvent, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Use executes one intent-level reuse request.
func (client *Client) Use(ctx context.Context, req *contract.UseRequest) (*contract.UseResponse, error) {
	var response contract.UseResponse
	if err := client.post(ctx, routeUses, req, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// UseStream executes one intent-level reuse request and reports live progress events.
func (client *Client) UseStream(ctx context.Context, req *contract.UseRequest, onEvent StreamHandler) (*contract.UseResponse, error) {
	var response contract.UseResponse
	if err := client.stream(ctx, routeUsesStream, req, onEvent, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Eval returns read-only acquisition and reuse metrics.
func (client *Client) Eval(ctx context.Context, req *contract.EvalRequest) (*contract.EvalResponse, error) {
	var response contract.EvalResponse
	if err := client.get(ctx, routeEval, evalQuery(req), &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func httpClient(base *http.Client, timeout time.Duration) *http.Client {
	if timeout == 0 {
		timeout = defaultTimeout
	}
	if base == nil {
		return &http.Client{Timeout: timeout}
	}
	copy := *base
	if timeout > 0 {
		copy.Timeout = timeout
	}
	return &copy
}
