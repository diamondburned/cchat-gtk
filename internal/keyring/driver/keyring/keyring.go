package keyring

import (
	"bytes"
	"encoding/gob"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/keyring/driver"
	"github.com/zalando/go-keyring"
)

type Provider struct{}

var _ driver.Provider = (*Provider)(nil)

func NewProvider() driver.Provider {
	return Provider{}
}

func (Provider) Get(service string, v interface{}) error {
	s, err := keyring.Get("cchat-gtk", service)
	if err != nil {
		return err
	}

	return gob.NewDecoder(strings.NewReader(s)).Decode(v)
}

func (Provider) Set(service string, v interface{}) error {
	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(v); err != nil {
		return err
	}

	return keyring.Set("cchat-gtk", service, b.String())
}
