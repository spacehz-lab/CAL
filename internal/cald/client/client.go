package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/cald"
	"github.com/spacehz-lab/cal/internal/cald/control"
	"github.com/spacehz-lab/cal/internal/calpath"
	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/discovery"
	caleval "github.com/spacehz-lab/cal/internal/eval"
	"github.com/spacehz-lab/cal/internal/runtime"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
	caluse "github.com/spacehz-lab/cal/internal/use"
)

const (
	defaultTimeout   = 5 * time.Minute
	discoveryTimeout = 20 * time.Minute
)

// Client calls the local cald HTTP API.
type Client struct {
	baseURL string
	http    *http.Client
}

// New resolves CAL home endpoint metadata and builds a local cald client.
func New(home string) (Client, error) {
	endpoint, err := readEndpoint(home)
	if err != nil {
		return Client{}, err
	}
	return NewForEndpoint(endpoint.BaseURL), nil
}

// NewForEndpoint builds a client for tests or already-discovered endpoints.
func NewForEndpoint(baseURL string) Client {
	return Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: defaultTimeout},
	}
}

// Status returns cald daemon status.
func (client Client) Status(ctx context.Context) (control.Status, error) {
	var status control.Status
	err := client.get(ctx, "/v1/daemon/status", &status)
	return status, err
}

// Stop asks cald to stop.
func (client Client) Stop(ctx context.Context) error {
	var ignored map[string]any
	return client.post(ctx, "/v1/daemon/stop", map[string]any{}, &ignored)
}

// AddProvider registers one provider path.
func (client Client) AddProvider(ctx context.Context, providerPath string) (core.Provider, error) {
	var response core.Provider
	err := client.post(ctx, "/v1/providers", map[string]string{"provider_path": providerPath}, &response)
	return response, err
}

// ListProviders returns stored Provider records.
func (client Client) ListProviders(ctx context.Context) ([]core.Provider, error) {
	var response struct {
		Providers []core.Provider `json:"providers"`
	}
	if err := client.get(ctx, "/v1/providers", &response); err != nil {
		return nil, err
	}
	return response.Providers, nil
}

// GetProvider returns one stored Provider record.
func (client Client) GetProvider(ctx context.Context, id string) (core.Provider, error) {
	var response core.Provider
	err := client.get(ctx, "/v1/providers/"+id, &response)
	return response, err
}

// GetProviderByPath returns a stored Provider record for one provider path.
func (client Client) GetProviderByPath(ctx context.Context, providerPath string) (core.Provider, error) {
	var response core.Provider
	query := url.Values{}
	query.Set("provider_path", providerPath)
	err := client.get(ctx, "/v1/providers/by-path?"+query.Encode(), &response)
	return response, err
}

// Discover runs synchronous provider acquisition.
func (client Client) Discover(ctx context.Context, req control.DiscoveryRequest) (discovery.JobResult, error) {
	var response discovery.JobResult
	err := client.withTimeout(discoveryTimeout).post(ctx, "/v1/discovery", req, &response)
	return response, err
}

// CapabilityListOptions filters capability summaries.
type CapabilityListOptions struct {
	CapabilityID string
	ProviderID   string
}

// ListCapabilities returns promoted capability summaries.
func (client Client) ListCapabilities(ctx context.Context, opts CapabilityListOptions) (runtime.CapabilityList, error) {
	var response runtime.CapabilityList
	path := "/v1/capabilities"
	query := url.Values{}
	if opts.CapabilityID != "" {
		query.Set("capability_id", opts.CapabilityID)
	}
	if opts.ProviderID != "" {
		query.Set("provider_id", opts.ProviderID)
	}
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	err := client.get(ctx, path, &response)
	return response, err
}

// GetCapability returns one full capability record.
func (client Client) GetCapability(ctx context.Context, id string) (core.Capability, error) {
	var response core.Capability
	err := client.get(ctx, "/v1/capabilities/"+id, &response)
	return response, err
}

// Run executes one promoted capability.
func (client Client) Run(ctx context.Context, req control.RunRequest) (core.Run, error) {
	var response core.Run
	err := client.post(ctx, "/v1/runs", req, &response)
	return response, err
}

// Use routes an intent to one promoted binding and executes it.
func (client Client) Use(ctx context.Context, req caluse.Request) (caluse.Result, error) {
	var response caluse.Result
	err := client.post(ctx, "/v1/uses", req, &response)
	return response, err
}

// GetRun returns one stored run record.
func (client Client) GetRun(ctx context.Context, id string) (core.Run, error) {
	var response core.Run
	err := client.get(ctx, "/v1/runs/"+id, &response)
	return response, err
}

// Eval returns aggregate CAL metrics.
func (client Client) Eval(ctx context.Context) (caleval.Metrics, error) {
	var response caleval.Metrics
	err := client.get(ctx, "/v1/eval", &response)
	return response, err
}

// GetTrace returns one persisted acquisition trace.
func (client Client) GetTrace(ctx context.Context, id string) (caltrace.Trace, error) {
	var response caltrace.Trace
	err := client.get(ctx, "/v1/traces/"+id, &response)
	return response, err
}

func (client Client) get(ctx context.Context, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+path, nil)
	if err != nil {
		return err
	}
	return client.do(req, target)
}

func (client Client) post(ctx context.Context, path string, body any, target any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return client.do(req, target)
}

func (client Client) do(req *http.Request, target any) error {
	resp, err := client.http.Do(req)
	if err != nil {
		return control.NewAPIError("cald_unavailable", err.Error())
	}
	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		apiErr := decodeAPIError(content)
		if apiErr.Code == "" {
			apiErr = control.NewAPIError("cald_error", strings.TrimSpace(string(content)))
		}
		return apiErr
	}
	if target == nil {
		return nil
	}
	if err := json.Unmarshal(content, target); err != nil {
		return fmt.Errorf("decode cald response: %w", err)
	}
	return nil
}

func (client Client) withTimeout(timeout time.Duration) Client {
	if client.http == nil {
		client.http = &http.Client{Timeout: timeout}
		return client
	}
	copiedHTTP := *client.http
	copiedHTTP.Timeout = timeout
	client.http = &copiedHTTP
	return client
}

func decodeAPIError(content []byte) control.APIError {
	var response struct {
		Error control.APIError `json:"error"`
	}
	_ = json.Unmarshal(content, &response)
	return response.Error
}

func readEndpoint(home string) (cald.EndpointFile, error) {
	resolved := strings.TrimSpace(home)
	if resolved == "" {
		var err error
		resolved, err = calpath.HomeDir()
		if err != nil {
			return cald.EndpointFile{}, err
		}
	} else {
		resolved = filepath.Clean(resolved)
	}
	content, err := os.ReadFile(cald.EndpointFilePath(resolved))
	if errors.Is(err, os.ErrNotExist) {
		return cald.EndpointFile{}, control.NewAPIError("cald_unavailable", "cald is not running")
	}
	if err != nil {
		return cald.EndpointFile{}, err
	}
	var endpoint cald.EndpointFile
	if err := json.Unmarshal(content, &endpoint); err != nil {
		return cald.EndpointFile{}, fmt.Errorf("decode cald endpoint: %w", err)
	}
	if strings.TrimSpace(endpoint.BaseURL) == "" {
		return cald.EndpointFile{}, control.NewAPIError("cald_unavailable", "cald endpoint is missing")
	}
	return endpoint, nil
}
