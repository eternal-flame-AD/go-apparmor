package apparmor

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindMountPoint(t *testing.T) {
	expect, err := shell(`grep -e "^securityfs" /proc/mounts | cut -d " " -f 2 | \
							 xargs -I{} sh -c '[ -d {}/apparmor ] && echo {}/apparmor'`)
	require.NoError(t, err)
	require.NotEmpty(t, expect)
	actual, err := findMountPoint()
	assert.NoError(t, err)
	assert.Equal(t, expect, actual)
}

func TestCheckEnabled(t *testing.T) {
	exp, _ := shell("aa-enabled")

	if exp != "Yes" {
		t.Skip("apparmor is not enabled")
	}

	enabled, err := AAIsEnabled()
	assert.NoError(t, err)
	assert.True(t, enabled)

}

func TestSplitConStatic(t *testing.T) {
	testCases := []string{
		"unconfined|unconfined||",
		"/path/to/executable (complain)|/path/to/executable|complain|",
		"/path/to/executable (enforce)|/path/to/executable|enforce|",
		"/path/to/executable//profile (complain)|/path/to/executable//profile|complain|",
	}
	for _, c := range testCases {
		parts := strings.Split(c, "|")
		label, mode, err := SplitCon(parts[0])
		assert.NoError(t, err)
		assert.Equal(t, parts[1], label)
		assert.Equal(t, parts[2], mode)
		assert.Equal(t, parts[3], "")
	}
}

func TestGetProcAttr(t *testing.T) {
	label, mode, err := GetProcAttr(os.Getpid(), "current")
	assert.NoError(t, err)
	assert.Equal(t, "unconfined", label)
	assert.Equal(t, "", mode)
}
