package apparmor

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"syscall"
	"testing"

	"github.com/eternal-flame-AD/go-apparmor/internal/apparmor_c"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestWithShim(t *testing.T) {
	if os.Getenv("APPARMOR_TEST_SHIM") != "1" {
		t.Skipf("APPARMOR_TEST_SHIM not set to 1, skipping")
	}
	shimModPath := "github.com/eternal-flame-AD/go-apparmor/cmd/go-apparmor-test-shim"
	shimExecName := "go-apparmor-test-shim"

	compile := func() string {
		// Compile the shim
		cmd := exec.Command("go", "build", "-o", shimExecName, shimModPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to compile shim: %v", err)
		}
		path, err := filepath.Abs(shimExecName)
		if err != nil {
			t.Fatalf("failed to get absolute path of shim: %v", err)
		}
		return path
	}

	shimExecPath := compile()
	defer os.Remove(shimExecPath)

	sockAddr := path.Join(os.TempDir(), "go-apparmor-test-shim.sock")
	defer os.Remove(sockAddr)
	sockListener, err := net.Listen("unix", sockAddr)
	if err != nil {
		t.Fatalf("failed to listen on socket: %v", err)
	}
	var sockConn *net.UnixConn

	runShim := func() (input *json.Encoder, output *json.Decoder) {
		cmd := exec.Command(shimExecPath, sockAddr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			t.Fatalf("failed to start shim: %v", err)
		}
		conn, err := sockListener.Accept()
		if err != nil {
			t.Fatalf("failed to accept connection: %v", err)
		}
		sockConn = conn.(*net.UnixConn)
		input, output = json.NewEncoder(conn), json.NewDecoder(conn)
		return
	}
	shimInput, shimOutput := runShim()

	writeCmd := func(cmd ShimCmd) {
		if err := shimInput.Encode(cmd); err != nil {
			t.Fatalf("failed to write command to shim: %v", err)
		}
	}

	readResp := func() ShimResponse {
		var resp ShimResponse
		if err := shimOutput.Decode(&resp); err != nil {
			t.Fatalf("failed to read response from shim: %v", err)
		}
		if !resp.Success {
			debug.PrintStack()
			t.Fatalf("shim returned error: %v", resp.ErrorStr)
		}
		return resp
	}

	writeCmd(ShimCmd{
		Command: "gettid",
	})
	resp := readResp()
	shimTid, err := strconv.Atoi(resp.Response[0])
	require.NoError(t, err)
	require.NotZero(t, shimTid)

	writeCmd(ShimCmd{
		Command: "getcon",
	})
	resp = readResp()
	labelExp, modeExp := resp.Response[0], resp.Response[1]
	require.Equal(t, "unconfined", labelExp, "process already confined, need cleanup?")
	label, mode, err := AAGetTaskCon(shimTid)
	assert.NoError(t, err)
	assert.Equal(t, labelExp, label)
	assert.Equal(t, modeExp, mode)
	/*
		rawconn, err := sockConn.SyscallConn()
		require.NoError(t, err)
		rawconn.Control(func(fd uintptr) {
			labelByConn, modeByConn, err := AAGetPeerCon(int(fd))
			assert.NoError(t, err)
			assert.Equal(t, labelExp, labelByConn)
			assert.Equal(t, modeExp, modeByConn)
		})
	*/

	writeCmd(ShimCmd{
		Command: "generate_profile",
		Args:    []string{shimExecPath, "test-profile", "test-hat", "test-subprofile", "complain"},
	})
	resp = readResp()
	profileStr := resp.Response[0]
	require.NotEmpty(t, profileStr)
	require.NoError(t, os.WriteFile("test-profile", []byte(profileStr), 0644))
	// defer os.Remove(resp.Response[1])

	profilePath, err := filepath.Abs("test-profile")
	require.NoError(t, err)
	os.Remove(".sudo-done")
	log.Println("now running sudo -S")
	sudoCmd := exec.Command("sudo", "-S", "apparmor_parser", "-r", profilePath)
	sudoCmd.Stdout = os.Stdout
	sudoCmd.Stderr = os.Stderr
	sudoCmd.Stdin = os.Stdin
	require.NoError(t, sudoCmd.Run())

	writeCmd(ShimCmd{
		Command: "exit",
	})
	sockConn.Close()

	testTransitions := func(usegoapi bool) {
		// start shim again
		shimInput, shimOutput = runShim()
		writeCmd(ShimCmd{
			Command: "gettid",
		})
		resp = readResp()
		shimTid, err = strconv.Atoi(resp.Response[0])
		require.NoError(t, err)
		require.NotZero(t, shimTid)

		writeCmd(ShimCmd{
			Command:  "getcon",
			UseGoAPI: usegoapi,
		})
		resp = readResp()
		labelExp, modeExp = resp.Response[0], resp.Response[1]
		require.Equal(t, "test-profile", labelExp)
		require.Equal(t, "complain", modeExp)
		label, mode, err = AAGetTaskCon(shimTid)
		assert.NoError(t, err)
		assert.Equal(t, labelExp, label)
		assert.Equal(t, modeExp, mode)

		sockConnF, err := sockConn.File()
		require.NoError(t, err)

		labelByConn, modeByConn, errGo := AAGetPeerCon(int(sockConnF.Fd()))
		labelC, modeC, errC := apparmor_c.AAGetPeerConC(int(sockConnF.Fd()))

		if errGo != errC {
			t.Fatalf("go and c error mismatch: %v != %v", errGo, errC)
		} else if errGo == nil {
			assert.NoError(t, err)
			assert.Equal(t, labelExp, labelC)
			assert.Equal(t, modeExp, modeC)

			assert.NoError(t, err)
			assert.Equal(t, labelExp, labelByConn)
			assert.Equal(t, modeExp, modeByConn)
		} else if errGo == syscall.ENOPROTOOPT {
			t.Logf("aa_getpeercon returned ENOPROTOOPT, skipping")
		} else {
			t.Fatalf("unexpected error: %v", errGo)
		}
		sockConnF.Close()

		writeCmd(ShimCmd{
			Command:  "change_hat",
			UseGoAPI: usegoapi,
			Args:     []string{"test-hat", "12345"},
		})
		_ = readResp()

		writeCmd(ShimCmd{
			Command: "getcon",
		})

		resp = readResp()
		labelExp, modeExp = resp.Response[0], resp.Response[1]
		require.Equal(t, "test-profile//test-hat", labelExp)
		require.Equal(t, "enforce", modeExp)
		label, mode, err = AAGetTaskCon(shimTid)
		assert.NoError(t, err)
		assert.Equal(t, labelExp, label)
		assert.Equal(t, modeExp, mode)

		writeCmd(ShimCmd{
			Command:  "change_hat",
			UseGoAPI: usegoapi,
			Args:     []string{"", "12345"},
		})
		_ = readResp()

		writeCmd(ShimCmd{
			Command: "getcon",
		})
		resp = readResp()
		labelExp, modeExp = resp.Response[0], resp.Response[1]
		require.Equal(t, "test-profile", labelExp)
		require.Equal(t, "complain", modeExp)
		label, mode, err = AAGetTaskCon(shimTid)
		assert.NoError(t, err)
		assert.Equal(t, labelExp, label)
		assert.Equal(t, modeExp, mode)

		writeCmd(ShimCmd{
			Command:  "change_profile",
			UseGoAPI: usegoapi,
			Args:     []string{"test-subprofile"},
		})
		_ = readResp()

		writeCmd(ShimCmd{
			Command: "getcon",
		})
		resp = readResp()
		labelExp, modeExp = resp.Response[0], resp.Response[1]
		require.Equal(t, "test-profile//null-test-subprofile", labelExp)
		require.Equal(t, "complain", modeExp)
		label, mode, err = AAGetTaskCon(shimTid)
		assert.NoError(t, err)
		assert.Equal(t, labelExp, label)
		assert.Equal(t, modeExp, mode)

		writeCmd(ShimCmd{
			Command: "exit",
		})
		sockConn.Close()
	}

	testTransitions(false)
	testTransitions(true)

}
