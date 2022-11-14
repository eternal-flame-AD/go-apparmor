package magic

import (
	"testing"

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
