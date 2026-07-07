package progress

import (
	"context"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestEmitCallsContextHandlerThenExplicitHandlers(t *testing.T) {
	var calls []string
	ctx := WithHandler(context.Background(), func(_ context.Context, event *model.ProgressEvent) {
		calls = append(calls, "context:"+string(event.Stage))
	})

	Emit(ctx, &model.ProgressEvent{Stage: model.ProgressStageProbe}, func(_ context.Context, event *model.ProgressEvent) {
		calls = append(calls, "explicit:"+string(event.Stage))
	})

	want := []string{"context:probe", "explicit:probe"}
	if len(calls) != len(want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls = %#v, want %#v", calls, want)
		}
	}
}

func TestWithHandlerComposesExistingContextHandler(t *testing.T) {
	var calls []string
	ctx := WithHandler(context.Background(), func(_ context.Context, event *model.ProgressEvent) {
		calls = append(calls, "first:"+string(event.Step))
	})
	ctx = WithHandler(ctx, func(_ context.Context, event *model.ProgressEvent) {
		calls = append(calls, "second:"+string(event.Step))
	})

	Emit(ctx, &model.ProgressEvent{Step: model.ProgressStepProposalBinding})

	want := []string{"first:binding", "second:binding"}
	if len(calls) != len(want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls = %#v, want %#v", calls, want)
		}
	}
}

func TestEmitAllowsNilHandlers(t *testing.T) {
	Emit(WithHandler(context.Background(), nil), &model.ProgressEvent{}, nil)
	Emit(context.Background(), nil, nil)
}
