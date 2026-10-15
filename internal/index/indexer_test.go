package index

import (
	"path/filepath"
	"testing"

	"github.com/hrishikeshs/sift/internal/db"
	"github.com/hrishikeshs/sift/internal/project"
)

func TestIndexerLegacyStructShapeAndDefaults(t *testing.T) {
	// Keep the original two-field positional literal source-compatible.
	_ = Indexer{nil, false}

	idx := &Indexer{}
	if err := idx.IndexAllProvider("unknown"); err == nil {
		t.Fatal("unknown provider should fail before touching discovery or the database")
	}
	if err := idx.IndexProjectProvider("/tmp/project", "unknown"); err == nil {
		t.Fatal("unknown project provider should fail")
	}
}

func TestProjectResult(t *testing.T) {
	if err := projectResult("/tmp/project", true, nil); err != nil {
		t.Fatal(err)
	}
	if err := projectResult("/tmp/project", false, nil); err == nil {
		t.Fatal("missing project should return the legacy not-found error")
	}
}

func TestIndexFileCodexEndToEnd(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "sift.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	idx := &Indexer{DB: database}
	fixture := filepath.Join("..", "parse", "testdata", "codex.jsonl")
	file := project.JSONLFile{
		Path: fixture, SessionID: "codex-session", ProjectHash: "codex:/tmp/example",
		ProjectPath: "/tmp/example", Provider: project.ProviderCodex,
	}
	count, err := idx.IndexFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("indexed %d Codex entries, want 3", count)
	}
	if again, err := idx.IndexFile(file); err != nil || again != 0 {
		t.Fatalf("incremental re-index = %d, %v; want 0, nil", again, err)
	}

	results, total, err := database.Search(db.SearchParams{
		Query: "boundary error", SourceType: "reasoning_summary",
	})
	if err != nil || total != 1 || len(results) != 1 {
		t.Fatalf("reasoning summary search = %#v, %d, %v", results, total, err)
	}
	var encrypted int
	if err := database.Conn().QueryRow(
		"SELECT COUNT(*) FROM entries WHERE content LIKE '%ENCRYPTED_REASONING%'",
	).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	if encrypted != 0 {
		t.Fatal("encrypted reasoning was persisted")
	}
}
