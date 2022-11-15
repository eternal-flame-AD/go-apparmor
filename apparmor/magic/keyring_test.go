package magic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyringStore(t *testing.T) {
	New := func() Store {
		s, err := NewKeyring(nil)
		require.NoError(t, err, "NewKeyring() should not return error")
		return s
	}
	testStore(t, New)
}

func TestKeyringStoreKeepAlive(t *testing.T) {
	sWithTimeout, err := NewKeyring(&KeyringOpts{TimeoutSeconds: 2})
	require.NoError(t, err, "NewKeyring() should not return error")
	sWithoutTimeout, err := NewKeyring(nil)
	require.NoError(t, err, "NewKeyring() should not return error")

	magic := uint64(0x12345678)
	require.NoError(t, sWithTimeout.Set(magic), "Set() should not return error")
	require.NoError(t, sWithoutTimeout.Set(magic), "Set() should not return error")

	start := time.Now()
	var timeoutErr error
	for time.Since(start) < 5*time.Second {
		m1, err := sWithTimeout.Get()
		if err != nil {
			timeoutErr = err
		} else {
			assert.Equal(t, magic, m1, "Get() should return the same magic")
		}
		m2, err := sWithoutTimeout.Get()
		require.NoError(t, err, "Get() should not return error")
		assert.Equal(t, magic, m2, "Get() should return the same magic")
	}
	assert.NotNil(t, timeoutErr, "Get() should return timeout error")
}
