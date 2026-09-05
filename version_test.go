// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// SchemaVersion must match the SPECVERSION the Makefile generates
// generated_types.go from.
func TestSchemaVersionMatchesMakefile(t *testing.T) {
	data, err := os.ReadFile("Makefile")
	require.NoError(t, err)

	m := regexp.MustCompile(`(?m)^SPECVERSION\s*:=\s*(\S+)\s*$`).FindSubmatch(data)
	require.NotNil(t, m, "SPECVERSION not found in Makefile")

	require.Equal(t, string(m[1]), SchemaVersion)
}
