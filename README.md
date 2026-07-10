# Sift

Full-text search for Claude Code and Codex sessions.

Index your local JSONL session transcripts into a SQLite database and search across agent responses, user messages, Claude thinking traces, and Codex reasoning summaries in milliseconds.

## Why Sift?

Claude Code and Codex write rich JSONL session transcripts that accumulate over weeks and months. These files:

- **Grow large** — a single session can reach hundreds of MB
- **Scatter** across `~/.claude/projects/` in UUID-named files
- **Get rotated** — once deleted, the context is gone forever

Sift solves this by indexing everything into a local SQLite+FTS5 database with daily compressed archives. Your conversations survive file rotation, and any thought from any session is a search away.

## Installation

```sh
go install github.com/hrishikeshs/sift@latest
```

Or build from source:

```sh
git clone https://github.com/hrishikeshs/sift.git
cd sift
go build -o sift .
```

## Quick Start

```sh
# Index all your Claude Code sessions
sift index --all

# Or opt in to Codex rollouts (Claude remains the default)
sift index --all --provider codex

# Index both providers
sift index --all --provider all

# Search for anything
sift search "heatmap rendering"

# See the conversation around a result
sift show 58611

# Set up automatic indexing (runs every hour)
sift install
```

## Search

```sh
# Simple search (multi-word queries auto-match as phrases)
sift search "auth middleware"

# Filter by time
sift search "dashboard" --since 2w
sift search "deployment" --since yesterday
sift search "bug fix" --since 2026-04-20

# Filter by content type
sift search "how should we" --type thinking    # agent reasoning
sift search "refactor" --type text             # agent responses
sift search "fix the bug" --type user          # your messages

# Filter by project
sift search "database migration" --project ~/workspace/myapp

# Filter by session
sift search "auth flow" --session abc12345-def6-7890-abcd-ef1234567890

# JSON output (for scripts and agents)
sift search "error handling" --json

# More results
sift search "testing" --limit 50
```

### Search Operators

Multi-word queries are automatically treated as phrase matches. For advanced queries:

```sh
# Explicit phrase (same as default for multi-word)
sift search '"exact phrase match"'

# Individual word matching with operators
sift search "auth AND migration"
sift search "redis OR memcached"
sift search "database NOT postgres"
```

## Show

Every search result includes an entry ID and a hint:

```
[2026-05-15 14:23] thinking | my-project #58611
  ...the heatmap component needs to re-render when...
  ~/.claude/projects/-Users-.../9620f8…2c.jsonl  →  sift show 58611
```

Run `sift show <id>` to see the full conversation context — 5 entries before and after:

```sh
sift show 58611

# More context
sift show 58611 --context 10
```

The target entry is highlighted with `◀` so you can spot it in the conversation flow.

## Indexing

### Manual

```sh
# Index everything
sift index --all

# Select a provider (claude is the backwards-compatible default)
sift index --all --provider codex
sift index --all --provider all

# Index a specific project
sift index --project ~/workspace/myapp

# Verbose output
sift index --all --verbose
```

### Automatic

```sh
# Install as a background service (runs every hour)
sift install

# Remove the service
sift uninstall

# Or run the daemon manually in the foreground
sift daemon --interval 30m
```

On macOS, `sift install` creates a LaunchAgent. On Linux, a systemd user service.

### What Gets Indexed

| Content | Indexed? | Type |
|---------|----------|------|
| Claude thinking traces | Yes | `thinking` |
| Codex reasoning summaries | Yes | `reasoning_summary` |
| Codex encrypted/raw reasoning | No | — |
| Agent text responses | Yes | `text` |
| Your messages | Yes | `user` |
| Tool names (Read, Edit, etc.) | Yes | `tool_use` |
| Tool results (file contents, CLI output) | No | — |
| Progress/hook/event messages | No | — |
| File history snapshots | No | — |

### Incremental & Efficient

- Tracks byte offsets per file — only reads new content since last index
- Unchanged files are skipped entirely (mtime + size check)
- First full index: ~60 seconds for a large dataset
- Subsequent runs: under 1 second

### Daily Archives

JSONL files can be rotated by the provider. Sift archives message records into daily gzipped chunks in the database, so your conversations survive file deletion. Tool results and Codex reasoning records (including encrypted reasoning) are excluded from archives.

## Housekeeping

```sh
# View index statistics
sift status

# Find and clean up entries from deleted files
sift gc --dry-run    # preview
sift gc              # execute
```

## How It Works

```
~/.claude/projects/ or ~/.codex/sessions/  ~/.sift/sift.db
├── -Users-...-myapp/                      ┌─────────────────────┐
│   ├── abc123.jsonl  ──── index ────────▶ │ entries (FTS5)      │
│   └── def456.jsonl  ──── index ────────▶ │ entries_fts         │
├── -Users-...-other/                      │ index_state         │
│   └── ...           ──── index ────────▶ │ chunks (gzip blobs) │
└── ...                                    └─────────────────────┘
                                                    │
                                           sift search "query"
                                                    │
                                                    ▼
                                           ranked results with
                                           highlighted snippets
```

1. **Discover** — scans `~/.claude/projects/` by default, or Codex's `sessions/` and `archived_sessions/` beneath `${CODEX_HOME:-~/.codex}` when selected
2. **Parse** — extracts provider-specific user and agent content; only explicit Codex reasoning summaries are indexed
3. **Index** — inserts into SQLite with FTS5 full-text search (porter stemming)
4. **Archive** — compresses raw JSONL lines into daily gzip chunks
5. **Search** — FTS5 ranked query with snippet highlighting

## Architecture

```
sift/
├── main.go
├── cmd/
│   ├── root.go           # Global flags
│   ├── index.go          # sift index
│   ├── search.go         # sift search
│   ├── show.go           # sift show
│   ├── status.go         # sift status
│   ├── gc.go             # sift gc
│   ├── daemon.go         # sift daemon
│   ├── install.go        # sift install
│   └── uninstall.go      # sift uninstall
├── internal/
│   ├── db/               # SQLite + FTS5 storage
│   ├── index/            # JSONL scanner + archival
│   ├── parse/            # Entry parsing + time duration
│   ├── project/          # Claude and Codex discovery
│   └── output/           # Terminal formatting
├── go.mod
└── go.sum
```

### Dependencies

- [cobra](https://github.com/spf13/cobra) — CLI framework
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — pure Go SQLite (no CGO)

Zero runtime dependencies. Single binary.

## Provider compatibility

Existing commands keep their original Claude behavior: omitting `--provider` is equivalent to `--provider claude`. Codex support is additive and does not change the database schema.

Codex rollout JSONL is a local implementation format and may evolve between CLI releases. Sift intentionally parses only a narrow set of records: `session_meta` for discovery and `response_item` messages/reasoning summaries for indexing. It never decodes or indexes `encrypted_content`, and ignores duplicate `event_msg` records.

## Database

Stored at `~/.sift/sift.db`. Override with `--db`:

```sh
sift --db /path/to/custom.db search "query"
```

## License

MIT
