# glance — LLM-optimized output summarizer

LLMs read every token of command output with equal attention, unlike humans who skim head/tail and scan for patterns. `glance` gives LLMs a human-like "skim" — pipe output in, get a token-efficient summary with head/tail + regex-matched lines, plus an ID to drill deeper.

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

- **POSIX sh, single file** — works everywhere including Alpine/busybox. No dependencies beyond standard Unix tools (`awk`, `cat`, `cp`, `date`, `mktemp`, `od`, `sed`, `stat`, `tr`, `wc`).
- **OR semantics** — all matchers (filters + presets) OR together. Head/tail always shown. All matching is case-insensitive. This is the most useful behavior for scanning output: "show me the start, end, and anything interesting".
- **Persistent storage** — captures stored in `$XDG_CACHE_HOME/glance/captures/` with timestamp + hex IDs (e.g. `20260219-143022-a3f8b1c0`). Full ID required for `glance show` — use `glance list` to find IDs.
- **No metadata files** — line count and age derived from the stored file itself (`wc -l`, `stat`).
- **Built-in + user presets** — three hardcoded presets (errors, warnings, status) cover common patterns. User presets in `~/.config/glance/presets.conf` for project-specific needs. Both use a unified sed-style format where the first character of each line is the delimiter (e.g. `/errors/error|err|fail/Error detection`). The delimiter is auto-picked to avoid conflicts with regex content.

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
| `glance presets add -d ',' <name> <re> [desc]` | Add with explicit delimiter |
| `glance presets remove <name>` | Remove user preset |

## Preset format

Presets use a sed-style format where the first character of each line is the delimiter:

```
/errors/error|err|fail|fatal|panic|exception|traceback/Error detection
/warnings/warn|warning|deprecated/Warnings
,pathpre,src/lib|test/unit,Path matching preset
```

When adding a preset, the delimiter is auto-picked from `/ , @ # % ~ !` — the first character not present in the regex. Use `-d` to override:

```
$ glance presets add mypreset 'error|fail' 'My errors'     # auto-picks /
$ glance presets add -d ',' pathpre 'src/lib' 'Paths'       # explicit , since regex has /
```

Preset names must be alphanumeric (plus hyphens and underscores). Comments (`#`) and blank lines are preserved in the user config file.

## Testing

```
./test_glance.sh
```

116 integration tests covering core pipe behavior, filtering, output format, show subcommand, list/clean, presets management, preset format handling, regex compatibility, age formatting, and edge cases (empty input, binary data, long lines, concurrency, flag validation, delimiter exhaustion).

## Files

- `glance` — the tool (~685 lines of POSIX sh)
- `test_glance.sh` — integration test suite
- `notes.md` — development notes
- `README.md` — this file
