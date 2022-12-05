//go:build linux
package apparmor

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"syscall"
)

// libraries/libapparmor/src/kernel.c

func gettid() int {
	return syscall.Gettid()
}

// findMountPoint find where the apparmor interface filesystem is mounted
func findMountPoint() (string, error) {
	mounts, err := os.Open("/proc/mounts")
	if err != nil {
		return "", err
	}
	defer mounts.Close()
	scanner := bufio.NewScanner(mounts)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.SplitN(line, " ", 4)
		if len(fields) >= 3 && fields[2] == "securityfs" /* mnt_type*/ {
			proposedPath := path.Join(fields[1], "apparmor")
			if stat, err := os.Stat(proposedPath); err == nil && stat.IsDir() {
				return proposedPath, nil
			}
		}
	}
	return "", errors.New("apparmor not loaded or not using securityfs")
}

type ParamBase string

const (
	ParamBaseEnabled        ParamBase = "enabled"
	ParamBasePrivateEnabled ParamBase = "available"
)

// CheckBase return if param is enabled in kernel module
func CheckBase(param ParamBase) (bool, error) {
	paramFile := path.Join("/sys/module/apparmor/parameters/", string(param))
	if f, err := os.Open(paramFile); err == nil {
		defer f.Close()
		var buf [2]byte
		if n, err := f.Read(buf[:]); err == nil || err == io.EOF {
			if n == 0 {
				return false, io.EOF
			}
			return buf[0] == 'Y', nil
		} else {
			return false, err
		}
	} else {
		return false, err
	}
}

// AAEnabled return if apparmor is enabled in kernel module
func AAIsEnabled() (bool, error) {
	return CheckBase(ParamBaseEnabled)
}

var (
	regexpConfinement = regexp.MustCompile(`^(.+)\s+\((.+)\)$`)
)

// SplitCon split a confinement string into label and mode
func SplitCon(confinement string) (label string, mode string, err error) {
	if confinement == "" {
		return "", "", errors.New("parameter is empty")
	}
	confinement = strings.TrimSuffix(confinement, "\n")
	if confinement == "unconfined" {
		return "unconfined", "", nil
	}

	if matches := regexpConfinement.FindStringSubmatch(confinement); matches != nil {
		return matches[1], matches[2], nil
	}
	return "", "", errors.New("unknown confinement format")
}

// GetProcAttrRaw return the raw confinement string of a process, read from /proc/attr
func GetProcAttrRaw(pid int, attr string) (string, error) {
	procAttrNew := path.Join("/proc", fmt.Sprint(pid), "attr/apparmor", attr)
	procAttrOld := path.Join("/proc", fmt.Sprint(pid), "attr", attr)
	if r, err := os.ReadFile(procAttrNew); err == nil {
		return string(r), nil
	} else if os.IsNotExist(err) {
		if r, err := os.ReadFile(procAttrOld); err == nil {
			return string(r), nil
		} else {
			return "", err
		}
	} else {
		return "", err
	}
}

// GetProcAttr return the confinement string of a process, split into label and mode
func GetProcAttr(pid int, attr string) (label string, mode string, err error) {
	if r, err := GetProcAttrRaw(pid, attr); err == nil {
		return SplitCon(r)
	} else {
		return "", "", err
	}
}

// SetProcAttr set the confinement string of a process
func SetProcAttr(pid int, attr string, con string) error {
	procAttrNew := path.Join("/proc", fmt.Sprint(pid), "attr/apparmor", attr)
	procAttrOld := path.Join("/proc", fmt.Sprint(pid), "attr", attr)
	if err := os.WriteFile(procAttrNew, []byte(con), 0); err == nil {
		return nil
	} else if os.IsNotExist(err) {
		if err := os.WriteFile(procAttrOld, []byte(con), 0); err == nil {
			return nil
		} else {
			return err
		}
	} else {
		return err
	}
}
