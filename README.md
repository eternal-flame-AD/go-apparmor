# go-apparmor

[![Go Reference](https://pkg.go.dev/badge/github.com/eternal-flame-AD/go-apparmor/apparmor.svg)](https://pkg.go.dev/github.com/eternal-flame-AD/go-apparmor/apparmor)

AppArmor 3.x kernel interface bindings written in pure Go.

## Examples

This package provides three levels of abstraction for transitioning between hats:

### Fully Managed using `apparmor.WithHat()`

The following example runs a function in a confined hat environment.

All necessary setup and teardown of Go the goroutine runtime is handled by the package.

```go
import (
    "github.com/eternal-flame-AD/go-apparmor/apparmor"
	"github.com/eternal-flame-AD/go-apparmor/apparmor/magic"
)

func ExampleWithHat() {
	// setup a place to store the magic key for transitioning back to the original hat
	keyRing, err := magic.NewKeyring(nil)
	if err != nil {
		log.Fatalf("failed to initialize kernel keyring: %v", err)
	}
	if generatedMagic, err := magic.Generate(nil); err != nil {
		log.Fatalf("failed to generate magic key: %v", err)
	} else if err := keyRing.Set(generatedMagic); err != nil {
		log.Fatalf("failed to store magic key: %v", err)
	}
	getMagic := func() uint64 {
		ret, err := keyRing.Get()
		if err != nil {
			log.Panicf("failed to get magic key: %v", err)
		}
		return ret
	}

	apparmor.WithHat("confined_hat", getMagic, func() {
		// This code is running in the confined_hat hat.

		// to spawn a goroutine:
		go apparmor.WithHat("confined_hat", getMagic, func() {
			// This code is ensured to be running in the confined_hat hat as well.
		})

	})
}

```

### Managed Setup using `apparmor.SetHat()`

The following example transitions a goroutine into a confined hat environment.
The caller is responsible for cleaning up by transitioning back to the original hat
or killing the thread by returning without calling `runtime.UnlockOSThread()`.

```go
func ExampleSetHat() {
	// setup a place to store the magic key for transitioning back to the original hat
	keyRing, err := magic.NewKeyring(nil)
	if err != nil {
		log.Fatalf("failed to initialize kernel keyring: %v", err)
	}
	if generatedMagic, err := magic.Generate(nil); err != nil {
		log.Fatalf("failed to generate magic key: %v", err)
	} else if err := keyRing.Set(generatedMagic); err != nil {
		log.Fatalf("failed to store magic key: %v", err)
	}
	getMagic := func() uint64 {
		ret, err := keyRing.Get()
		if err != nil {
			log.Panicf("failed to get magic key: %v", err)
		}
		return ret
	}
	go func() {
		apparmor.SetHat("confined_hat", getMagic())

		// This code is running in the confined_hat hat.
		// it is NOT safe to spawn goroutines without additional setup here.

		// if you are able to transition the thread back to the original state,
		// call runtime.UnlockOSThread() as many times as you called apparmor.SetHat().
		// otherwise, the underlying OS thread will not be reused and will be killed by the scheduler.
	}()
}

```

### Unmanaged Using libapparmor style functions


Functions that begin with `AA` are functional replicas of their respective libapparmor C API.
That means the caller is responsible for setting up the goroutine runtime so that the OS thread
that gets modified is scheduled to the desired goroutine(s), this usually boils down to:
- Locking the thread to the calling goroutine with `runtime.LockOSThread()`
- Make sure that additional goroutines that are spawned are also locked to a transitioned thre.
- Determining whether the thread could be reused by calling `runtime.UnlockOSThread()` after transitioning the thread back to its original state.


```go
func getConfinement() {
    label, mode, err := apparmor.AAGetCon()
    if err != nil {
        log.Fatalf("failed to get confinement: %v, do we have introspect permissions?", err)
    }
    log.Printf("confinement: %s, mode: %s", label, mode)
    // example: confinement: /usr/bin/program, mode: enforce
}

func transitionToSubprofile() {
    go func() {
        runtime.LockOSThread()
        // IMPORTANT: do not unlock the thread unless you can transition the OS thread
        // back to its original profile. Otherwise, other goroutines may be confined
        // to the subprofile as well.

        apparmor.AAChangeProfile("sub-profile")

        exec.Command("cat", "/proc/self/attr/apparmor/current").Run()
        // example: .../sub-profile
    }()
   
}
```