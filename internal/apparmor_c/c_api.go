package apparmor_c

import (
	// #cgo LDFLAGS: -lapparmor
	// #include "./apparmor.h"
	"C"
)
import (
	"syscall"
	"unsafe"
)

func AAChangeHatC(subprofile string, magicToken uint64) error {
	var ret uintptr
	if subprofile != "" {
		subProfileC := C.CString(subprofile)
		defer C.free(unsafe.Pointer(subProfileC))
		ret = uintptr(C.go_aa_change_hat(subProfileC, C.ulong(magicToken)))
	} else {
		ret = uintptr(C.go_aa_change_hat(nil, C.ulong(magicToken)))
	}

	if ret != 0 {
		return syscall.Errno(ret)
	}
	return nil
}

func AAChangeProfileC(subprofile string) error {
	var ret uintptr

	if subprofile != "" {
		subProfileC := C.CString(subprofile)
		defer C.free(unsafe.Pointer(subProfileC))
		ret = uintptr(C.go_aa_change_profile(subProfileC))
	} else {
		ret = uintptr(C.go_aa_change_profile(nil))
	}

	if ret != 0 {
		return syscall.Errno(ret)
	}
	return nil
}

func AAGetConfinementC() (label, mode string, err error) {
	var labelC, modeC *C.char
	ret := uintptr(C.go_aa_getcon(&labelC, &modeC))
	if ret != 0 {
		return "", "", syscall.Errno(ret)
	}
	defer C.free(unsafe.Pointer(labelC))
	// mode is in the same buffer so we don't need to free it
	return C.GoString(labelC), C.GoString(modeC), nil
}

func AAChangeHatVC(subprofiles []string, magicToken uint64) error {
	var ret uintptr
	if len(subprofiles) > 0 {
		subProfilesC := C.malloc(C.size_t(len(subprofiles)+1) * C.size_t(unsafe.Sizeof(uintptr(0))))
		subProfiles := (*[1 << 30]*C.char)(subProfilesC)
		for i, subprofile := range subprofiles {
			subProfiles[i] = C.CString(subprofile)
			defer C.free(unsafe.Pointer(subProfiles[i]))
		}
		subProfiles[len(subprofiles)] = nil
		ret = uintptr(C.go_aa_change_hatv((**C.char)(unsafe.Pointer(&subProfiles[0])), C.ulong(magicToken)))
	} else {
		ret = uintptr(C.go_aa_change_hatv(nil, C.ulong(magicToken)))
	}

	if ret != 0 {
		return syscall.Errno(ret)
	}
	return nil
}

func AAGetPeerConC(fd int) (label, mode string, err error) {
	var labelC, modeC *C.char
	ret := uintptr(C.go_aa_getpeercon(C.int(fd), &labelC, &modeC))
	if ret != 0 {
		return "", "", syscall.Errno(ret)
	}
	defer C.free(unsafe.Pointer(labelC))
	// mode is in the same buffer so we don't need to free it
	return C.GoString(labelC), C.GoString(modeC), nil
}
