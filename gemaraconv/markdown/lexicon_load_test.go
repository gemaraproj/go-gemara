package markdown

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/fetcher"
	"github.com/stretchr/testify/require"
)

func TestParseLexiconYAML_golden(t *testing.T) {
	entries, err := parseLexiconYAML(readLexiconTestdata(t, "lexicon_good.yaml"))
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
	entries, err := loadLexiconFromURI(context.Background(), &fetcher.URI{}, lexiconTestdataAbsPath(t, "lexicon_good.yaml"))
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

// TestLexiconGistIntegration fetches a live gist, so it only runs when the
// caller explicitly opts in via GEMARA_NETWORK_TESTS=1. The default test run
// (local and CI) must never touch the network; the HTTP fetch path itself is
// covered offline by TestLoadLexiconFromURI_http below.
func TestLexiconGistIntegration(t *testing.T) {
	if os.Getenv("GEMARA_NETWORK_TESTS") != "1" {
		t.Skip("network test: set GEMARA_NETWORK_TESTS=1 to run")
	}
	const gist = "https://gist.githubusercontent.com/eddie-knight/3ffa5e1a5d562ba0f3b0cd3f5b563679/raw/1b39cb516f23430288b7893004eb0b91f14a7487/lexicon.yaml"
	entries, err := loadLexiconFromURI(context.Background(), &fetcher.URI{}, gist)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
}

// TestLoadLexiconFromURI_http exercises the http(s) fetch path against a local
// httptest.Server, keeping that branch covered without external dependence.
func TestLoadLexiconFromURI_http(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(readLexiconTestdata(t, "lexicon_good.yaml"))
		require.NoError(t, err)
	}))
	t.Cleanup(srv.Close)

	// gemara.Load detects the format from the extension, so the URL needs a
	// .yaml path; the server serves the same document for every path.
	entries, err := loadLexiconFromURI(context.Background(), &fetcher.URI{}, srv.URL+"/lexicon.yaml")
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "Example Term", entries[0].Canonical)
}
