package gemaraconv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/stretchr/testify/require"
)

func readLexiconTestdata(t *testing.T, name string) []byte {
	t.Helper()
	fileBytes, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return fileBytes
}

func lexiconFileURL(t *testing.T, name string) string {
	t.Helper()
	absPath, err := filepath.Abs(filepath.Join("testdata", name))
	require.NoError(t, err)
	return "file://" + filepath.ToSlash(absPath)
}

func TestParseLexiconYAML_golden(t *testing.T) {
	entries, err := parseLexiconYAML(readLexiconTestdata(t, "lexicon_golden.yaml"))
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "Example Term", entries[0].Canonical)
	require.Contains(t, entries[0].Definition, "example term")
	require.Equal(t, []string{"ET", "sample term"}, entries[0].Synonyms)
	require.Len(t, entries[0].Refs, 1)
	require.Equal(t, "Example spec", entries[0].Refs[0].Citation)
	require.Equal(t, "https://example.com/docs/example-term", entries[0].Refs[0].URL)
	require.Equal(t, "Second Term", entries[1].Canonical)
	require.Empty(t, entries[1].Refs)
}

func TestParseLexiconYAML_rejects(t *testing.T) {
	cases := []string{
		"lexicon_empty_terms.yaml",
		"lexicon_list_root.yaml",
		"lexicon_bad_term.yaml",
		"lexicon_bad_ref.yaml",
		"lexicon_dup_canonical.yaml",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := parseLexiconYAML(readLexiconTestdata(t, name))
			require.Error(t, err)
		})
	}
}

func TestLoadLexiconFromURI_file(t *testing.T) {
	entries, err := loadLexiconFromURI(lexiconFileURL(t, "lexicon_golden.yaml"))
	require.NoError(t, err)
	require.Len(t, entries, 2)
}

func TestResolveLexiconURL(t *testing.T) {
	meta := gemaraMetadataWithLexicon("lex", "https://example.com/lex.yaml")
	resolvedURL, err := resolveLexiconURL(meta)
	require.NoError(t, err)
	require.Equal(t, "https://example.com/lex.yaml", resolvedURL)
}

func TestResolveLexiconURL_remarksFallback(t *testing.T) {
	meta := gemara.Metadata{
		Lexicon: &gemara.ArtifactMapping{
			ReferenceId: "missing",
			Remarks:     "https://gist.example/raw/lex.yaml",
		},
	}
	resolvedURL, err := resolveLexiconURL(meta)
	require.NoError(t, err)
	require.Equal(t, "https://gist.example/raw/lex.yaml", resolvedURL)
}

func TestResolveLexiconURL_errors(t *testing.T) {
	_, err := resolveLexiconURL(gemara.Metadata{})
	require.Error(t, err)

	meta := gemara.Metadata{
		Lexicon: &gemara.ArtifactMapping{ReferenceId: "x"},
	}
	_, err = resolveLexiconURL(meta)
	require.Error(t, err)
}

func gemaraMetadataWithLexicon(refID, lexURL string) gemara.Metadata {
	return gemara.Metadata{
		Lexicon: &gemara.ArtifactMapping{ReferenceId: refID},
		MappingReferences: []gemara.MappingReference{
			{Id: refID, Title: "L", Version: "1", Url: lexURL},
		},
	}
}

func TestLexiconGistIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("network")
	}
	const gist = "https://gist.githubusercontent.com/eddie-knight/3ffa5e1a5d562ba0f3b0cd3f5b563679/raw/1b39cb516f23430288b7893004eb0b91f14a7487/lexicon.yaml"
	entries, err := loadLexiconFromURI(gist)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
}
