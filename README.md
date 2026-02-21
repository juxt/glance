# glance — LLM-optimized output summarizer

LLMs read every token of command output with equal attention, unlike humans who skim head/tail and scan for patterns. `glance` gives LLMs a human-like "skim" — pipe output in, get a token-efficient summary with head/tail + regex-matched lines, plus an ID to drill deeper.

## Install

```sh
# Run without installing
uvx glancecli

# Install permanently
uv tool install glancecli

# Or with Go
go install github.com/akeboshiwind/glance@latest
```

## What it does

```
$ seq 500 | glance -n 3
  1: 1
  2: 2
  3: 3
498: 498
499: 499
500: 500
--- glance id=20260219-143022-a3f8b1c0 | 500 lines | showing 6 | sections: 1-3, 498-500 ---
```

```
$ (seq 100; echo "ERROR db connection refused"; seq 200) | glance -n 3 -p errors
  1: 1
  2: 2
  3: 3
101: ERROR db connection refused
299: 198
300: 199
301: 200
--- glance id=20260219-143025-b7c2e4f1 | 301 lines | showing 7 | sections: 1-3, 101, 299-301 ---
```

The stored capture can be drilled into:
```
$ glance show 20260219-143025-b7c2e4f1 -a 101 2
 99: 99
100: 100
101: ERROR db connection refused
102: 1
103: 2
--- glance show 20260219-143025-b7c2e4f1 | 301 lines | showing 5 | sections: 99-103 ---
```

## Design decisions

- **Single static binary** — compiled Go, no runtime dependencies. Cross-compiled for Linux, macOS, and Windows (amd64 + arm64).
- **OR semantics** — all matchers (filters + presets) OR together. Head/tail always shown. This is the most useful behavior for scanning output: "show me the start, end, and anything interesting".
- **Persistent storage** — captures stored in `$XDG_CACHE_HOME/glance/captures/` with timestamp + hex IDs (e.g. `20260219-143022-a3f8b1c0`). Full ID required for `glance show` — use `glance list` to find IDs.
- **No metadata files** — line count and age derived from the stored file itself (`wc -l`, `stat`).
- **Built-in + user presets** — three hardcoded presets (errors, warnings, status) cover common patterns. User presets stored in `~/.config/glance/presets.csv` as CSV. Use `(?i)` prefix for case-insensitive matching.

## Usage

See `glance help` for complete documentation. Key commands:

| Command | Description |
|---------|-------------|
| `cmd \| glance` | Head 10 + tail 10 |
| `cmd \| glance -n 5` | Head 5 + tail 5 |
| `cmd \| glance -f 'regex'` | + regex filter matches |
| `cmd \| glance -p errors` | + preset filter |
| `glance show <id>` | Full stored output |
| `glance show <id> -l 50-80` | Line range |
| `glance show <id> -f 'regex'` | Filter stored output |
| `glance show <id> -p errors` | Filter with preset |
| `glance show <id> -a N C` | Context around line N |
| `glance list` | List stored captures |
| `glance clean` | Purge captures |
| `glance presets list` | Show all presets |
| `glance presets add <name> <re> [desc]` | Add user preset |
| `glance presets remove <name>` | Remove user preset |


Preset names must be alphanumeric (plus hyphens and underscores).

## Testing

```
go test ./...
```

## Files

- `main.go` — entry point and CLI
- `*_test.go` — integration and unit tests
- `.github/workflows/publish.yml` — PyPI release workflow
- `README.md` — this file
