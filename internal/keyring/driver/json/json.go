package json

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

var ErrUnsafePermission = errors.New("secrets.json file has unsafe permission")

type Provider struct {
	dir string
}

func NewProvider(dir string) Provider {
	return Provider{dir}
}

func (p Provider) open(service string, write bool) (*os.File, error) {
	var flags int
	if write {
		// If file does not exist, then create a new one. Else, erase the file.
		// The file can only be written onto.
		flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	} else {
		// If file does not exist, then error out. The file can only be read
		// from.
		flags = os.O_RDONLY
	}

	// Make a filename using the given service.
	var filename = fmt.Sprintf("%s_secret.json", SanitizeName(service))

	f, err := os.OpenFile(filepath.Join(p.dir, filename), flags, 0600)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open file")
	}

	// Stat the file and verify that the permissions are safe.
	s, err := f.Stat()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to stat file")
	}
	if m := s.Mode(); m != 0600 {
		return nil, fmt.Errorf("secrets.json file has unsafe permission %06o", m)
	}

	return f, nil
}

// Get unmarshals the service from the JSON secret file. It errors out if the
// secret file does not exist.
func (p Provider) Get(service string, v interface{}) error {
	f, err := p.open(service, false)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(v); err != nil {
		return errors.Wrap(err, "Failed to decode JSON")
	}

	return nil
}

// Set writes the service's data into a JSON secret file.
func (p Provider) Set(service string, v interface{}) error {
	f, err := p.open(service, true)
	if err != nil {
		return err
	}
	defer f.Close()

	// Create a new formatted encoder.
	enc := json.NewEncoder(f)
	enc.SetIndent("", "\n")

	// Encode using created encoder.
	if err := enc.Encode(v); err != nil {
		return errors.Wrap(err, "Failed to encode JSON")
	}

	return nil
}

// SanitizeName sanitizes the name so that it's safe to use as a filename.
func SanitizeName(name string) string {
	// Escape all weird characters in a filename.
	name = strings.Map(underscoreNonAlphanum, name)

	// Lower-case everything.
	name = strings.ToLower(name)

	return name
}

func underscoreNonAlphanum(r rune) rune {
	// especially does not cover slashes
	if unicode.IsLetter(r) || unicode.IsNumber(r) {
		return r
	}
	return '_'
}
