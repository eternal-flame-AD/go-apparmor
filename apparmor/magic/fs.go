package magic

import (
	"os"
	"strconv"
)

// FS implements Store using a file on the filesystem.
// The file is created with 0600 permissions.
// The Apparmor hat should be denied access to the file.
type FS struct {
	filename string
}

func (f *FS) Set(magic uint64) error {
	if err := os.WriteFile(f.filename, []byte(strconv.FormatUint(magic, 16)), 0600); err != nil {
		return err
	}
	return nil
}

func (f *FS) Get() (uint64, error) {
	if data, err := os.ReadFile(f.filename); err != nil {
		return 0, err
	} else {
		return strconv.ParseUint(string(data), 16, 64)
	}
}

func (f *FS) Clear() error {
	if err := os.Remove(f.filename); !os.IsNotExist(err) {
		return err
	}
	return nil
}

func NewFS(filename string) (Store, error) {
	fs := &FS{filename: filename}
	return fs, nil
}
