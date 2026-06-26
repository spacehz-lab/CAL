package cald

import "github.com/spacehz-lab/cal/internal/cald/control"

// Status describes local cald availability.
type Status = control.Status

// LocalStatus returns local cald status without starting a service.
func LocalStatus() Status {
	return Status{
		Running: false,
		Mode:    "local",
		Message: "cald HTTP service is not running",
	}
}
