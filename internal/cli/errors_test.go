package cli

import "testing"

func TestNewCommandErrorf(t *testing.T) {
	err := newCommandErrorf(commandErrorTargetProviderNotFound, "target %q missing", "/tmp/fake")
	if err.Code != string(commandErrorTargetProviderNotFound) || err.Message != `target "/tmp/fake" missing` {
		t.Fatalf("newCommandErrorf() = %#v, want formatted command error", err)
	}
}
