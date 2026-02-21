---
name: glance
description: Use when about to run commands with potentially long output (>40 lines) — builds, tests, logs, kubectl, docker, grep, find. Pipes output through glance for token-efficient summaries instead of reading all tokens.
version: 0.1.0
allowed-tools: Bash(glance *)
---

## Bootstrap

Check if glance is available:

```sh
which glance
```

If missing, use `uvx --from glancecli glance` as the prefix for all glance commands (e.g. `cmd 2>&1 | uvx --from glancecli glance`).

**First time:** run `glance help` to learn the full interface.

## When to use

Pipe through glance when output is unknown or likely >40 lines:
- Build output (`go build`, `npm run build`, `cargo build`, `make`)
- Test output (`go test`, `pytest`, `jest`, `cargo test`)
- Logs (`kubectl logs`, `docker logs`, `journalctl`)
- Container/cluster commands (`kubectl get`, `docker ps`)
- Large searches (`grep -r`, `find`, `rg`) — pipe through glance first, then drill into the captured results instead of re-running with different flags

## When to skip

Commands with reliably short output — `git status`, `pwd`, `which`, `echo`, `ls` (small dirs), `git log -5`.

## Workflow

### Skim

```sh
cmd 2>&1 | glance
# or with a preset for builds/tests:
cmd 2>&1 | glance -p errors
```

Read the footer — it gives you the capture ID and line count.

### Drill

Don't jump to full output. Use targeted queries first:

```sh
glance show <id> -a LINE 5    # context around a specific line
glance show <id> -l 50-80     # line range
glance show <id> -f 'regex'   # filter stored output
```

### Save-and-query

Pipe once, then query the stored capture multiple times with different filters/ranges — including with different tools. Avoids re-running expensive commands. Especially useful for broad exploratory searches — run one wide `rg` or `find`, capture it, then use `glance show <id> -f` to explore different aspects of the results.

If you lose track of the capture ID, use `glance list` to find it.

## Key gotcha

Always use `2>&1` — without it, stderr (where most errors go) isn't captured by the pipe.
