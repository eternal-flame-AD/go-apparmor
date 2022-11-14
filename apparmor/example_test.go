package apparmor_test

import (
	"log"
	"runtime"

	"github.com/eternal-flame-AD/go-apparmor/apparmor"
	"github.com/eternal-flame-AD/go-apparmor/apparmor/magic"
)

func ExampleAAChangeHat() {

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

	// make sure the goroutine runs in its dedicated task
	runtime.LockOSThread()

	// transition to ^confined_hat
	if err := apparmor.AAChangeHat("confined_hat", getMagic()); err != nil {
		log.Panicf("failed to change hat: %v", err)
	}

	confined_code := func() {
		// This code is running in the confined_hat hat.
		// it is NOT safe to spawn goroutines here.
	}
	confined_code()

	// transition back to the original hat
	if err := apparmor.AAChangeHat("", getMagic()); err != nil {
		log.Panicf("failed to change hat: %v", err)
	}

	// Do not use defer() because it may cause the panicked thread to be reused.
	// Go scheduler will ensure the thread is killed if this goroutine panics.
	runtime.UnlockOSThread()
}

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
