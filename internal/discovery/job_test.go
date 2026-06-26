package discovery

import (
	"testing"
	"time"
)

func TestNewJobIDUsesUTCUnixNano(t *testing.T) {
	now := time.Unix(0, 42).In(time.FixedZone("test", 3600))
	if got := newJobID(now); got != "disc_42" {
		t.Fatalf("newJobID() = %q, want disc_42", got)
	}
}
