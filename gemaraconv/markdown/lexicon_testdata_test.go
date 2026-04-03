package markdown

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// lexiconTestdataAbsPath returns an absolute path to go-gemara/test-data/<name> (tests run with cwd gemaraconv/markdown/).
func lexiconTestdataAbsPath(t *testing.T, name string) string {
	t.Helper()
	absPath, err := filepath.Abs(filepath.Join("..", "..", "test-data", name))
	require.NoError(t, err)
	return absPath
}

func readLexiconTestdata(t *testing.T, name string) []byte {
	t.Helper()
	fileBytes, err := os.ReadFile(lexiconTestdataAbsPath(t, name))
	require.NoError(t, err)
	return fileBytes
}

func lexiconFileURL(t *testing.T, name string) string {
	t.Helper()
	return "file://" + filepath.ToSlash(lexiconTestdataAbsPath(t, name))
}
