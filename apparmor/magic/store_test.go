package magic

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func testStore(t *testing.T, New func() Store) {
	s := New()
	err := s.Clear()
	require.NoError(t, err, "Store.Clear() should not return error unless keyring is not available")

	magic, err := s.Get()
	require.Equal(t, magic, uint64(0))
	require.Error(t, err, "Store.Get() should return error if key is not set")

	for reuse := 0; reuse < 10; reuse++ {
		magicTest := rand.Uint64()
		err = s.Set(magicTest)
		require.NoError(t, err, "Store.Set() should not return error")

		for tries := 0; tries < 10; tries++ {
			magic, err = s.Get()
			require.NoError(t, err, "Store.Get() should not return error")
			require.Equal(t, magic, uint64(magicTest))
		}
	}
}
