package llm

import (
	"errors"
	"testing"
)

func TestOptionsValidateTrimsAndRequiresFields(t *testing.T) {
	opts := Options{
		API:     APIResponses,
		BaseURL: " http://127.0.0.1 ",
		Model:   " test-model ",
		APIKey:  " test-key ",
	}
	if err := opts.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if opts.BaseURL != "http://127.0.0.1" || opts.Model != "test-model" || opts.APIKey != "test-key" {
		t.Fatalf("Validate() = %#v, want trimmed options", opts)
	}
}

func TestOptionsValidateRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name string
		opts *Options
		want error
	}{
		{name: "nil", opts: nil, want: ErrMissingOptions},
		{name: "missing api", opts: &Options{APIKey: "test-key", Model: "test-model"}, want: ErrMissingAPI},
		{name: "missing key", opts: &Options{API: APIResponses, Model: "test-model"}, want: ErrMissingAPIKey},
		{name: "missing model", opts: &Options{API: APIResponses, APIKey: "test-key"}, want: ErrMissingModel},
		{name: "unsupported api", opts: &Options{API: "unknown", APIKey: "test-key", Model: "test-model"}, want: ErrUnsupportedAPI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if !errors.Is(err, tt.want) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestNewSelectsConfiguredAPI(t *testing.T) {
	for _, api := range []API{APIResponses, APIChatCompletions} {
		client, err := New(&Options{API: api, APIKey: "test-key", Model: "test-model"})
		if err != nil {
			t.Fatalf("New(%s) error = %v", api, err)
		}
		if client.Model() != "test-model" {
			t.Fatalf("Model() = %q, want test-model", client.Model())
		}
	}
}

func TestNewRejectsMissingOptions(t *testing.T) {
	if _, err := New(nil); !errors.Is(err, ErrMissingOptions) {
		t.Fatalf("New(nil) error = %v, want ErrMissingOptions", err)
	}
}
