package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/spacehz-lab/cal/internal/contract"
)

func (client *Client) get(ctx context.Context, path string, query url.Values, target any) error {
	return client.do(ctx, http.MethodGet, path, query, nil, target)
}

func (client *Client) post(ctx context.Context, path string, body any, target any) error {
	return client.do(ctx, http.MethodPost, path, nil, body, target)
}

func (client *Client) do(ctx context.Context, method string, path string, query url.Values, body any, target any) error {
	req, err := client.newRequest(ctx, method, path, query, body)
	if err != nil {
		return err
	}
	resp, err := client.http.Do(req)
	if err != nil {
		return newError(0, contract.ErrorCaldUnavailable, err.Error())
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return decodeErrorResponse(resp.StatusCode, data)
	}
	if target == nil {
		return nil
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (client *Client) newRequest(ctx context.Context, method string, path string, query url.Values, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, client.url(path, query), reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func (client *Client) url(path string, query url.Values) string {
	full := client.baseURL + "/" + strings.TrimLeft(path, "/")
	if len(query) == 0 {
		return full
	}
	return full + "?" + query.Encode()
}

func capabilityQuery(req *contract.CapabilityListRequest) url.Values {
	if req == nil {
		return nil
	}
	return idQuery(req.CapabilityID, req.ProviderID)
}

func evalQuery(req *contract.EvalRequest) url.Values {
	if req == nil {
		return nil
	}
	return idQuery(req.CapabilityID, req.ProviderID)
}

func idQuery(capabilityID string, providerID string) url.Values {
	query := url.Values{}
	if capabilityID != "" {
		query.Set(queryCapabilityID, capabilityID)
	}
	if providerID != "" {
		query.Set(queryProviderID, providerID)
	}
	return query
}
