package cald

import "testing"

func TestLocalStatus(t *testing.T) {
	status := LocalStatus()
	if status.Running {
		t.Fatal("LocalStatus().Running = true, want false")
	}
	if status.Mode != "local" {
		t.Fatalf("LocalStatus().Mode = %q, want local", status.Mode)
	}
}
