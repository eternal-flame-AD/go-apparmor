//go:build linux
package apparmor

import (
	"errors"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

func AAGetPeerCon(fd int) (label string, mode string, err error) {
	bufSize := uint32(64)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var rawPeerCon string

	maxTries := 5
	for i := 0; i < maxTries; i++ {

		buf := make([]byte, bufSize)

		bufSizePtr := unsafe.Pointer(&bufSize)

		_, _, err := syscall.Syscall6(
			syscall.SYS_GETSOCKOPT, uintptr(fd), syscall.SOL_SOCKET, syscall.SO_PEERSEC,
			uintptr(unsafe.Pointer(&buf[0])), uintptr(bufSizePtr), 0)
		// bufSize should have updated value now

		if err == 0 {
			rawPeerCon = string(buf[:bufSize])
			rawPeerCon = strings.TrimSuffix(rawPeerCon, "\x00")
			return SplitCon(rawPeerCon)
		} else if err != syscall.ERANGE {
			return "", "", err
		}

		runtime.KeepAlive(buf)
		runtime.KeepAlive(bufSize)

		// just to be sure
		bufSize++

	}
	return "", "", errors.New("too many retries but buffer is still too small")
}
