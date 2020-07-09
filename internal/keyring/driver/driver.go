package driver

import (
	"fmt"

	"github.com/diamondburned/cchat-gtk/internal/log"
)

type Provider interface {
	Get(service string, v interface{}) error
	Set(service string, v interface{}) error
}

type Store struct {
	providers []Provider
}

func NewStore(providers ...Provider) Store {
	return Store{providers}
}

func (s Store) Get(service string, v interface{}) error {
	for _, provider := range s.providers {
		if err := provider.Get(service, v); err == nil {
			return nil
		} else {
			log.Info(err)
		}
	}
	return fmt.Errorf("service %s not found in keyring services.", service)
}

func (s Store) Set(service string, v interface{}) error {
	for _, provider := range s.providers {
		if err := provider.Set(service, v); err == nil {
			return nil
		} else {
			log.Info(err)
		}
	}
	return fmt.Errorf("failed to set keyring for service %s", service)
}
