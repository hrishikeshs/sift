package parse

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractEntriesClaudeCompatibility(t *testing.T) {
	lines := fixtureLines(t, "claude.jsonl")

	user := ExtractEntries(lines[0], "claude.jsonl", "/tmp/example", "hash", 12)
	if len(user) != 1 || user[0].SourceType != "user" || user[0].Content != "Please fix the parser." {
		t.Fatalf("unexpected Claude user entries: %#v", user)
	}

	assistant := ExtractEntries(lines[1], "claude.jsonl", "/tmp/example", "hash", 34)
	if len(assistant) != 3 {
		t.Fatalf("got %d Claude assistant entries, want 3: %#v", len(assistant), assistant)
	}
	wantTypes := []string{"thinking", "text", "tool_use"}
	for i, want := range wantTypes {
		if assistant[i].SourceType != want {
			t.Errorf("entry %d type = %q, want %q", i, assistant[i].SourceType, want)
		}
	}
}

func TestExtractCodexEntries(t *testing.T) {
	lines := fixtureLines(t, "codex.jsonl")
	var gotTypes, gotContent []string
	for i, line := range lines {
		entries := ExtractCodexEntries(line, "codex.jsonl", "/tmp/example", "codex:/tmp/example", "codex-session", int64(i*100))
		for _, entry := range entries {
			gotTypes = append(gotTypes, entry.SourceType)
			gotContent = append(gotContent, entry.Content)
			if entry.SessionID != "codex-session" || entry.ProjectPath != "/tmp/example" {
				t.Errorf("Codex metadata not retained: %#v", entry)
			}
		}
	}

	wantTypes := []string{"user", "reasoning_summary", "text"}
	if len(gotTypes) != len(wantTypes) {
		t.Fatalf("Codex types = %#v, want %#v", gotTypes, wantTypes)
	}
	for i, want := range wantTypes {
		if gotTypes[i] != want {
			t.Errorf("Codex type %d = %q, want %q", i, gotTypes[i], want)
		}
		if gotContent[i] == "ENCRYPTED_REASONING_MUST_NOT_BE_INDEXED" {
			t.Fatal("encrypted reasoning was indexed")
		}
	}
}

func TestCodexArchiveWorthyExcludesReasoning(t *testing.T) {
	lines := fixtureLines(t, "codex.jsonl")
	want := []bool{false, true, false, true, false}
	for i, line := range lines {
		if got := CodexArchiveWorthy(line); got != want[i] {
			t.Errorf("line %d archive-worthy = %v, want %v", i, got, want[i])
		}
	}
}

func fixtureLines(t *testing.T, name string) [][]byte {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var lines [][]byte
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, append([]byte(nil), scanner.Bytes()...))
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return lines
}
