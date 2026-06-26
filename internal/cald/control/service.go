package control

import (
	"strings"

	"github.com/spacehz-lab/cal/internal/config"
	calstore "github.com/spacehz-lab/cal/internal/store"
)

// Service owns cald control-plane workflows shared by HTTP and future adapters.
type Service struct {
	store *calstore.Store
	cfg   *config.File
}

// NewService opens CAL-home-backed control dependencies.
func NewService(home string) (Service, error) {
	var store *calstore.Store
	var err error
	if strings.TrimSpace(home) == "" {
		store, err = calstore.OpenFromEnv()
	} else {
		store, err = calstore.Open(home)
	}
	if err != nil {
		return Service{}, err
	}
	if err := store.Ensure(); err != nil {
		return Service{}, err
	}
	return Service{
		store: store,
		cfg:   config.New(store.Home()),
	}, nil
}

// Home returns the service CAL home.
func (svc Service) Home() string {
	return svc.store.Home()
}
