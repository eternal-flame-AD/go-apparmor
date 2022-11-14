package magic

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFSStore(t *testing.T) {
	name := path.Join(os.TempDir(), fmt.Sprintf("test-magic-fs-%d", rand.Uint64()))
	New := func() Store {
		s, err := NewFS(name)
		require.NoError(t, err, "NewKeyring() should not return error")
		return s
	}
	testStore(t, New)
}
