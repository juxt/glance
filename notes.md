# glance — development notes

## What is this?
LLM-optimized output summarizer. Pipe command output in, get token-efficient head/tail + regex-matched lines, plus an ID to drill deeper.

## Design choices
- POSIX sh, single file, works everywhere
- Storage in XDG cache dir
- All matchers OR together, head/tail always shown
- Built-in presets hardcoded, user presets in config file

## Progress
- [x] Created folder structure
- [x] Implemented full glance script (~350 lines POSIX sh)
- [x] All features: pipe mode, head/tail, -n, -f, -p, --no-store
- [x] Subcommands: show (full, --lines, --filter, --around), list, clean, presets, help
- [x] 47/47 integration tests passing
- [x] README.md report

## Learnings
- `printf '--- ...'` fails in some shells because format string starting with `-` gets parsed as option. Use `printf '%s\n' "--- ..."` instead.
- `wc -l` counts newlines, so `printf '%s' "text"` (no trailing newline) gives count off by one. Always use `printf '%s\n'` when piping to `wc -l`.
- POSIX sh has no arrays — used newline-separated strings with `IFS` manipulation for built-in presets.
- `grep -E` doesn't support `(?i)` for case-insensitive. Used `grep -iE` instead.
- Cross-platform `stat` for mtime: try GNU `stat -c %Y` first, then macOS `stat -f %m`, then fallback.

## Issues found (subagent review, 2026-02-19)

Three subagents reviewed the project: code reviewer, new-user tester, agent needle-in-haystack.

### Critical
1. [x] **O(n²) sed loop** — fixed. Replaced per-line `sed -n Np` with single-pass awk in do_pipe and sed range + awk in do_show --around.
2. [x] **`wc -l` undercounts** — fixed. Replaced `wc -l` with `awk END { print NR }` which counts records regardless of trailing newline.
3. [x] **No `-n` validation** — fixed. Reject non-numeric values before processing input.
4. [x] **`timing` preset too broad** — fixed. Removed the preset entirely; users can add their own via `glance presets add`.

### Major
5. [-] **Partial ID matching missing** — cancelled. Full IDs required by design. Updated help and README examples to use full IDs.
6. [-] **No gap indicator** — cancelled. Line numbers make gaps obvious; adding `...` wastes tokens, counter to design goals.

### Minor
7. [x] **"1 lines" grammar** — fixed. Added plural_lines helper for singular/plural in headers.
8. [-] **Help examples use short IDs** — fixed alongside #5, examples now use full IDs.
9. [-] **TOCTOU race** — cancelled. Timestamp + 8-char random hex makes collisions near-impossible. Removed the unnecessary retry loop.
10. [x] **No validation on `--lines` range format** — fixed. Reject non-numeric values before passing to sed.

## Unified sed-style preset format (2026-02-20)

Replaced pipe-delimited built-in and tab-delimited user preset formats with sed-style format where first char is delimiter. Added `parse_preset_line` helper, auto-delimiter selection, `-d` flag.

## Test & feature gaps to address (TDD)

### Test gaps (already passing, just needed tests)
1. [x] Comments and blank lines preserved through add/remove
2. [x] Preset update (re-add same name) replaces old entry
3. [x] Unknown preset with `-p` produces error
4. [x] `presets add` with missing args shows usage
5. [x] `clean --all` removes user presets
8. [x] `--around` near boundaries (line 1, last line)
9. [x] `--lines` and `--filter` long-form flags in pipe mode
10. [x] Description containing delimiter round-trips correctly

### Feature fixes (needed code changes, TDD red-green)
6. [x] `-n 0` should error — added `0` to rejection case pattern
7. [x] Missing value for `-f`/`-p` — added `[ $# -ge 2 ]` guards
11. [x] Preset name validation — added alphanumeric+hyphen+underscore check

82/82 tests passing.

## Subagent review (2026-02-20)

Six agents reviewed the code: bug hunter, general reviewer, conciseness, test coverage, adversarial fuzzer, docs audit. Deduplicated findings below.

### Bugs

1. [x] **`-d` with delimiter present in regex silently corrupts** — fixed. Added validation rejecting delimiter if present in regex.
2. [x] **Preset names starting with `-` conflict with flags** — fixed. Reject names starting with `-`.
3. [x] **`-f` patterns starting with `-` passed as grep flags** — fixed. Added `--` before user patterns in grep calls.
4. [x] **`--around` no validation on center/context args** — fixed. Added numeric validation for both args.
5. [-] **`--lines` in show accepts single number** — skipped. Low priority, harmless behavior.

### Security

6. [x] **Path traversal in `glance show`** — fixed. Validate capture ID rejects `/` and `..`.

### Docs

7. [x] **Show flags mutually exclusive** — fixed. Rewrote do_show to OR all flags together, matching pipe mode behavior.
8. [x] **Case-insensitive matching undocumented** — fixed. Added note to help text.
9. [x] **Hardcoded `~/.config` in `glance help presets`** — fixed. Uses dynamic `$GLANCE_CONFIG` path.

### Conciseness

10. [x] **Filter accumulation verbose** — fixed. Used `${var:+|}` idiom, saved 8 lines.
11. [-] **Range-building in do_pipe** — skipped. Already shorter after refactors; shell loops are clear enough.
12. [-] **Trap updated 3 times** — already resolved. Only one trap after refactors.
13. [-] **Trivial helpers could inline** — skipped. Names communicate intent, used in multiple places.

### Missing tests

14. [-] **Help output** — added then dropped. Smoke tests were too weak to be useful.
15. [x] **Invalid regex** — added test. awk errors on invalid regex, test confirms.
16. [x] **Age formatting boundaries** — added 7 boundary tests (0s, 59s, 60s, 3599s, 3600s, 86399s, 86400s).
17. [x] **No subcommand for presets** — added test. Confirms usage message.
18. [x] **Delimiter exhaustion** — added test. Confirms error when all candidates used.

### Additional fixes

- [x] **Built-in presets used `\s`** — awk doesn't support Perl-style `\s`. Changed to POSIX `[[:space:]]`.
- [x] **Help text hardcoded preset list** — now prints dynamically from `BUILTIN_PRESETS`.
- [x] **Displaced `format_age` comment** — moved to correct location.
- [x] **Shared `print_matched_lines` function** — unified pipe and show into single-pass awk.
- [x] **Summary moved from header to footer** — enables single-pass output.

109/109 tests passing.
