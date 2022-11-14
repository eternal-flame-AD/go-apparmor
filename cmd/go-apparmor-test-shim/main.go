package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/eternal-flame-AD/go-apparmor/apparmor"
	"github.com/eternal-flame-AD/go-apparmor/internal/apparmor_c"
)

type ShimCmd struct {
	Command  string   `json:"command"`
	UseGoAPI bool     `json:"use-go-api"`
	Args     []string `json:"args"`
}

type ShimResponse struct {
	Success  bool   `json:"success"`
	ErrorStr string `json:"error"`

	Response []string `json:"response"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s <socket>", os.Args[0])
	}
	conn, err := net.Dial("unix", os.Args[1])
	if err != nil {
		log.Fatalf("failed to connect to socket: %v", err)
	}
	defer conn.Close()
	input := json.NewDecoder(conn)
	output := json.NewEncoder(conn)
	for {
		var cmd ShimCmd
		if err := input.Decode(&cmd); err != nil {
			log.Panicf("cannot decode input: %v", err)
		}
		var resp ShimResponse

		switch {
		case cmd.Command == "gettid":
			resp.Response = []string{strconv.Itoa(os.Getpid())}
			resp.Success = true
		case cmd.Command == "getcon":
			if cmd.UseGoAPI {
				label, mode, err := apparmor.AAGetCon()
				if err != nil {
					resp.ErrorStr = err.Error()
				} else {
					resp.Response = []string{label, mode}
					resp.Success = true
				}
			} else {
				label, mode, err := apparmor_c.AAGetConfinementC()
				if err != nil {
					resp.ErrorStr = err.Error()
				} else {
					resp.Response = []string{label, mode}
					resp.Success = true
				}
			}
		case cmd.Command == "generate_profile":
			ctx := profileTplContext{
				ExecPath:       cmd.Args[0],
				ExecDotPath:    strings.TrimPrefix(strings.ReplaceAll(cmd.Args[0], "/", "."), "."),
				ProfileName:    cmd.Args[1],
				HatName:        cmd.Args[2],
				SubProfileName: cmd.Args[3],
				Flags:          cmd.Args[4],
			}
			var outBuf bytes.Buffer
			err := profileTpl.Execute(&outBuf, ctx)
			if err != nil {
				resp.ErrorStr = err.Error()
			} else {
				resp.Response = []string{outBuf.String(), ctx.ExecDotPath}
				resp.Success = true
			}
		case cmd.Command == "change_hat":
			magicToken, err := strconv.ParseUint(cmd.Args[1], 10, 64)
			if err != nil {
				resp.ErrorStr = err.Error()
			}
			if cmd.UseGoAPI {
				err = apparmor.AAChangeHat(cmd.Args[0], magicToken)
				if err != nil {
					resp.ErrorStr = err.Error()
				} else {
					resp.Success = true
				}
			} else {
				err = apparmor_c.AAChangeHatC(cmd.Args[0], magicToken)
				if err != nil {
					resp.ErrorStr = err.Error()
				} else {
					resp.Success = true
				}
			}
		case cmd.Command == "change_hat_v":
			magicToken, err := strconv.ParseUint(cmd.Args[0], 10, 64)
			if err != nil {
				resp.ErrorStr = err.Error()
			}
			if cmd.UseGoAPI {
				/*
					err = apparmor.AAChangeHatV(cmd.Args[1:], magicToken)
					if err != nil {
						resp.ErrorStr = err.Error()
					} else {
						resp.Success = true
					}*/
				resp.ErrorStr = "not implemented"
			} else {
				err = apparmor_c.AAChangeHatVC(cmd.Args[1:], magicToken)
				if err != nil {
					resp.ErrorStr = err.Error()
				} else {
					resp.Success = true
				}
			}
		case cmd.Command == "change_profile":
			if cmd.UseGoAPI {
				err := apparmor.AAChangeProfile(cmd.Args[0])
				if err != nil {
					resp.ErrorStr = err.Error()
				} else {
					resp.Success = true
				}
			} else {
				err := apparmor_c.AAChangeProfileC(cmd.Args[0])
				if err != nil {
					resp.ErrorStr = err.Error()
				} else {
					resp.Success = true
				}
			}
		case cmd.Command == "exit":
			os.Exit(0)
		default:
			resp.ErrorStr = "unknown command"
			resp.Success = false
		}

		if err := output.Encode(&resp); err != nil {
			log.Panicf("cannot encode output: %v", err)
		}
	}
}
