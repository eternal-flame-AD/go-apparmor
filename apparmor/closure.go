package apparmor

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// SetHat locks the current goroutine by calling runtime.LockOSThread(),
// compares the current hat by AAGetCon() with hat and changes the hat if necessary.
//
// runtime.LockOSThread() is called regardless of the return value
// The caller is responsible for determining whether and when runtime.UnlockOSThread()
// should be called.
//
// Generally, the caller should only call runtime.UnlockOSThread() if the
// goroutine is transitioned out of the hat and the underlying OS thread is reusable.
// Otherwise, the caller should let the scheduler kill the thread by
// returning without calling runtime.UnlockOSThread().
func SetHat(hat string, magic uint64) error {
	runtime.LockOSThread()
	if hat == "" {
		if err := AAChangeHat("", magic); err != nil {
			return err
		}
		return nil
	}

	curLabel, _, err := AAGetCon()
	if err != nil {
		return fmt.Errorf("cannot get current label: %v", err)
	}

	if !strings.HasSuffix(curLabel, "//"+hat) {
		if err := AAChangeHat(hat, magic); err != nil {
			return err
		}

		curLabel, _, err = AAGetCon()
		if err != nil {
			return err
		}
		if !strings.HasSuffix(curLabel, "//"+hat) {
			return fmt.Errorf("failed to change hat to %q", hat)
		}
	}

	return nil
}

// WithHat spawns achanges the current hat to hat and then calls fn. It restores the
// original hat when fn returns.
//
// Magic is a function that when called should return the same non-zero value as the
// magic token for transitioning out of the hat
//
// For higher level of protection, do not save this token in process memory during
// the execution of fn.
//
// If you want to spawn new goroutines in the hat, use go armor.WithHat(...) instead.
// WithHat() will compare the scheduled goroutine's hat with the current hat and
// transition if necessary.
//
// For performance consideration, do not spawn a large number of goroutines with
// WithHat() as each will need its own OS-backed thread.
//
// WithHat will return an error if the hat cannot be changed, fn() will not be executed.
// If you want to leave a hat (i.e. pass in empty string), use SetHat() instead.
func WithHat(hat string, magic func() uint64, fn func()) error {
	wait := make(chan error)

	if hat == "" {
		return errors.New("WithHat() does not accept empty string, are you trying to use SetHat()?")
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				if err, ok := err.(error); ok {
					wait <- err
					return
				}
				wait <- fmt.Errorf("panic: %v", err)
			}
		}()

		if err := SetHat(hat, magic()); err != nil {
			wait <- err
			return
		}

		fn()

		if err := SetHat("", magic()); err != nil {
			wait <- err
			return
		}
		// only if we transitioned back successfully could we reuse this thread
		// otherwise let the scheduler kill this thread
		runtime.UnlockOSThread()
		// call twice because we called SetHat() twice
		runtime.UnlockOSThread()
		wait <- nil

	}()
	return <-wait
}
