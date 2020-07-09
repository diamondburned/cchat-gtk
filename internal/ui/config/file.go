package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
)

// dirPath indicates the path to the config. This variable is created when
// __init is called.
var dirPath string

// Singleton to initialize the config directories once.
var __initonce sync.Once

func __init() {
	// Load the config dir:
	d, err := os.UserConfigDir()
	if err != nil {
		log.Fatalln("Failed to get config dir:", err)
	}

	// Fill Path:
	dirPath = filepath.Join(d, "cchat-gtk")

	// Ensure it exists:
	if err := os.Mkdir(dirPath, 0755|os.ModeDir); err != nil && !os.IsExist(err) {
		log.Fatalln("Failed to make config dir:", err)
	}
}

// DirPath returns the config directory.
func DirPath() string {
	// Ensure that files and folders are initialized.
	__initonce.Do(__init)

	return dirPath
}

// MarshalToFile marshals the given interface into the given filename. The
// filename will be prepended with the config directory.
func MarshalToFile(file string, from interface{}) error {
	file = filepath.Join(DirPath(), file)

	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_SYNC|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrap(err, "Failed to open file")
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")

	if err := enc.Encode(from); err != nil {
		return errors.Wrap(err, "Failed to marshal given struct")
	}

	return nil
}

// UnmarshalFromFile unmarshals the given filename to the given interface. The
// filename will be prepended with the config directory. IsNotExist errors are
// ignored.
func UnmarshalFromFile(file string, to interface{}) error {
	file = filepath.Join(DirPath(), file)

	f, err := os.OpenFile(file, os.O_RDONLY, 0644)
	if err != nil {
		// Ignore does not exist error, leave struct as it is.
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(to); err != nil {
		return errors.Wrap(err, "Failed to unmarshal to given struct")
	}

	return nil
}
