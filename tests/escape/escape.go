//go:build linux
package main

import (
	// #cgo LDFLAGS: -lc
	// #include <sched.h>
	// int sched_getcpu(void);
	"C"

	"runtime"
	"text/template"
)
import (
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eternal-flame-AD/go-apparmor/apparmor"
)

type TplContext struct {
	ProfileName string
	ExecPath    string
	TestDir     string
	ExecDotPath string
}

const ProfileName = "test-profile"

var tpl = template.Must(template.New("").Parse(`

include <tunables/global>

profile {{ .ProfileName }} {{.ExecPath}} flags=(enforce) {
	include <abstractions/base>
	include <abstractions/apparmor_api/introspect>
	
	{{.ExecPath}} mr,

	{{ .TestDir }}/protected_ro rw,

	include if exists <local/{{.ExecDotPath}}>

	^contained {
		include <abstractions/base>
		include <abstractions/apparmor_api/introspect>

		deny {{ .TestDir }}/protected_ro w,
		{{ .TestDir }}/protected_ro r,

		{{.ExecPath}} mr,	
	}
}
`))

func magic() uint64 {
	// for testing only
	return 1
}

func tryAppend(dir string, hat string, numGoRoutine int, shouldSuccess bool, duration time.Duration, wg *sync.WaitGroup) (cpuStat []uint64) {
	cpuUsed := make([]uint64, runtime.NumCPU())
	start := time.Now()

	wg.Add(numGoRoutine)
	for i := 0; i < numGoRoutine; i++ {

		fun := func() {
			defer wg.Done()
			for time.Now().Sub(start) < duration {
				cpuIdx := int(C.sched_getcpu())
				atomic.AddUint64(&cpuUsed[cpuIdx], 1)
				f, err := os.OpenFile(path.Join(dir, "protected_ro"), os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					if shouldSuccess {
						label, mode, err := apparmor.AAGetCon()
						if err != nil {
							panic(err)
						}
						log.Panicf("Failed to open file in safe context, label: %s, mode: %s, err: %v", label, mode, err)
					}
					continue
				} else {
					f.Close()
					if !shouldSuccess {
						label, mode, err := apparmor.AAGetCon()
						if err != nil {
							panic(err)
						}
						log.Panicf("Opened file in unsafe context, label: %s, mode: %s, err: %v", label, mode, err)
					}
				}
				runtime.Gosched()
			}
		}
		if hat != "" {
			go apparmor.WithHat(hat, magic, fun)
		} else {
			go fun()
		}
	}
	return cpuUsed
}

func main() {
	label, mode, err := apparmor.AAGetCon()
	if err != nil {
		panic(err)
	}
	if label == "unconfined" {
		tplCtx := TplContext{
			ProfileName: ProfileName,
			TestDir:     os.TempDir(),
		}
		tplCtx.ExecPath, err = os.Executable()
		if err != nil {
			panic(err)
		}
		tplCtx.ExecDotPath = strings.ReplaceAll(tplCtx.ExecPath, "/", ".")[1:]
		f, err := os.Create(ProfileName)
		if err != nil {
			panic(err)
		}
		if err := tpl.Execute(f, tplCtx); err != nil {
			panic(err)
		}
		log.Printf("Created profile %s", ProfileName)
	} else if label == ProfileName {
		if mode != "enforce" {
			log.Panicf("Wrong mode: %s", mode)
		}
		log.Printf("Running in profile %s", ProfileName)
		{
			if err := os.WriteFile(path.Join(os.TempDir(), "protected_ro"), []byte("test"), 0777); err != nil {
				panic(err)
			}
			wg := &sync.WaitGroup{}
			cpuStat1 := tryAppend(os.TempDir(), "", 100, true, 5*time.Second, wg)
			var cpuStat2 []uint64
			apparmor.WithHat("contained", magic, func() {
				cpuStat2 = tryAppend(os.TempDir(), "contained", 100, false, 5*time.Second, wg)
			})
			cpuStat3 := tryAppend(os.TempDir(), "", 100, true, 5*time.Second, wg)
			wg.Wait()
			log.Printf("cpuStat1: %v", cpuStat1)
			log.Printf("cpuStat2: %v", cpuStat2)
			log.Printf("cpuStat3: %v", cpuStat3)
		}

	} else {
		log.Panicf("Wrong label: %s", label)
	}
}
