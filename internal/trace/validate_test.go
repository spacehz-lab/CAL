package trace

import "testing"

func TestValidateRejectsInvalidStatus(t *testing.T) {
	if err := Validate(Trace{ID: "trace_abc", Status: "done"}); err == nil {
		t.Fatal("Validate() error = nil, want invalid status error")
	}
}

func TestValidateAllowsKnownStatuses(t *testing.T) {
	for _, status := range []Status{StatusRunning, StatusCompleted, StatusFailed, StatusCanceled} {
		if err := Validate(Trace{ID: "trace_abc", Status: status}); err != nil {
			t.Fatalf("Validate(%q) error = %v", status, err)
		}
	}
}

func TestValidateRequiresID(t *testing.T) {
	if err := Validate(Trace{Status: StatusRunning}); err == nil {
		t.Fatal("Validate() error = nil, want missing id error")
	}
}
