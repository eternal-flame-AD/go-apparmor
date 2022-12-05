//go:build linux
package magic

import (
	"crypto/rand"
	"fmt"
	"io"
)

// Generate is a convenience function that returns a magic token
// from a rendom source.
// crypto/rand is used if r == nil
func Generate(r io.Reader) (uint64, error) {
	if r == nil {
		r = rand.Reader
	}
	var buf [64 / 8]byte
	if _, err := r.Read(buf[:]); err != nil {
		return 0, fmt.Errorf("failed to generate magic token: %v", err)
	}
	return uint64(buf[0])<<56 | uint64(buf[1])<<48 |
		uint64(buf[2])<<40 | uint64(buf[3])<<32 |
		uint64(buf[4])<<24 | uint64(buf[5])<<16 |
		uint64(buf[6])<<8 | uint64(buf[7]), nil
}
