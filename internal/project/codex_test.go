package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverCodex(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)

	active := filepath.Join(home, "sessions", "2026", "07", "10", "rollout-2026-07-10T02-00-00-11111111-2222-3333-4444-555555555555.jsonl")
	archived := filepath.Join(home, "archived_sessions", "rollout-2026-07-09T02-00-00-aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.jsonl")
	for _, path := range []string{active, archived} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(active, []byte("{\"timestamp\":\"2026-07-10T02:00:00Z\",\"type\":\"session_meta\",\"payload\":{\"session_id\":\"active-session\",\"cwd\":\"/tmp/active\"}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archived, []byte("{\"timestamp\":\"2026-07-09T02:00:00Z\",\"type\":\"session_meta\",\"payload\":{\"id\":\"archived-session\",\"cwd\":\"/tmp/archived\"}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := DiscoverCodex()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2: %#v", len(files), files)
	}
	if files[0].Provider != ProviderCodex || files[1].Provider != ProviderCodex {
		t.Fatalf("provider metadata missing: %#v", files)
	}
	if files[0].SessionID != "archived-session" || files[0].ProjectPath != "/tmp/archived" {
		t.Errorf("unexpected archived metadata: %#v", files[0])
	}
	if files[1].SessionID != "active-session" || files[1].ProjectHash != "codex:/tmp/active" {
		t.Errorf("unexpected active metadata: %#v", files[1])
	}
}

func TestDiscoverCodexMissingHomeIsEmpty(t *testing.T) {
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "missing"))
	files, err := DiscoverCodex()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("got %#v, want no files", files)
	}
}
