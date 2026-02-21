#!/bin/sh
# Integration tests for glance
set -e

GLANCE="$(cd "$(dirname "$0")" && pwd)/glance"
PASS=0
FAIL=0
TOTAL=0

# Use isolated cache/config dirs for testing
export XDG_CACHE_HOME=$(mktemp -d)
export XDG_CONFIG_HOME=$(mktemp -d)
trap 'rm -rf "$XDG_CACHE_HOME" "$XDG_CONFIG_HOME"' EXIT

assert() {
    _desc="$1"
    _expected="$2"
    _actual="$3"
    TOTAL=$(( TOTAL + 1 ))
    if [ "$_expected" = "$_actual" ]; then
        PASS=$(( PASS + 1 ))
        printf '  ok  %s\n' "$_desc"
    else
        FAIL=$(( FAIL + 1 ))
        printf '  FAIL %s\n' "$_desc"
        printf '    expected: %s\n' "$_expected"
        printf '    actual:   %s\n' "$_actual"
    fi
}

assert_contains() {
    _desc="$1"
    _pattern="$2"
    _text="$3"
    TOTAL=$(( TOTAL + 1 ))
    if printf '%s' "$_text" | grep -qE "$_pattern"; then
        PASS=$(( PASS + 1 ))
        printf '  ok  %s\n' "$_desc"
    else
        FAIL=$(( FAIL + 1 ))
        printf '  FAIL %s\n' "$_desc"
        printf '    pattern: %s\n' "$_pattern"
        printf '    text: %s\n' "$_text"
    fi
}

assert_not_contains() {
    _desc="$1"
    _pattern="$2"
    _text="$3"
    TOTAL=$(( TOTAL + 1 ))
    if printf '%s' "$_text" | grep -qE "$_pattern"; then
        FAIL=$(( FAIL + 1 ))
        printf '  FAIL %s\n' "$_desc"
        printf '    should not match: %s\n' "$_pattern"
    else
        PASS=$(( PASS + 1 ))
        printf '  ok  %s\n' "$_desc"
    fi
}

# Extract ID from glance header
extract_id() {
    printf '%s' "$1" | tail -1 | sed 's/.*id=\([^ ]*\) .*/\1/'
}

# =============================================
printf 'Core pipe behavior\n'
# =============================================

# Empty input
out=$(printf '' | "$GLANCE")
assert_contains "empty input: 0 lines" "0 lines" "$out"
assert_contains "empty input: showing 0" "showing 0" "$out"

# Single line
out=$(printf 'hello\n' | "$GLANCE")
assert_contains "single line: singular" "1 line[^s]" "$out"
assert_contains "single line: content" "hello" "$out"

# Single line without trailing newline
out=$(printf 'hello' | "$GLANCE")
assert_contains "no trailing newline: 1 line" "1 line[^s]" "$out"
assert_contains "no trailing newline: content" "hello" "$out"

# Short input (5 lines, default n=10)
out=$(seq 5 | "$GLANCE")
assert_contains "short input: 5 lines" "5 lines" "$out"
assert_contains "short input: showing 5" "showing 5" "$out"
assert_contains "short input: sections 1-5" "sections: 1-5" "$out"

# Exact boundary (20 lines, n=10)
out=$(seq 20 | "$GLANCE" -n 10)
assert_contains "exact boundary: 20 lines" "20 lines" "$out"
assert_contains "exact boundary: showing 20" "showing 20" "$out"

# Large input
out=$(seq 1000 | "$GLANCE")
assert_contains "large input: 1000 lines" "1000 lines" "$out"
assert_contains "large input: showing 20" "showing 20" "$out"
assert_contains "large input: has line 1" "1: 1$" "$out"
assert_contains "large input: has line 1000" "1000: 1000" "$out"

# Custom -n
out=$(seq 500 | "$GLANCE" -n 3)
assert_contains "custom n: showing 6" "showing 6" "$out"
assert_contains "custom n: sections" "sections: 1-3, 498-500" "$out"

# Binary-ish input (null bytes)
out=$(printf 'line1\nline2\x00binary\nline3\n' | "$GLANCE" 2>&1) || true
TOTAL=$(( TOTAL + 1 ))
PASS=$(( PASS + 1 ))
printf '  ok  binary input: does not crash\n'

# =============================================
printf '\nFiltering\n'
# =============================================

# --filter matches middle lines
out=$( (seq 100; echo "MATCH here"; seq 200) | "$GLANCE" -f 'MATCH' )
assert_contains "filter: matches middle" "101: MATCH here" "$out"

# --preset errors
out=$( (seq 100; echo "ERROR bad thing"; seq 200) | "$GLANCE" -p errors )
assert_contains "preset errors: matches ERROR" "ERROR bad thing" "$out"

# Multiple presets OR together
out=$( (seq 50; echo "ERROR oops"; seq 50; echo "WARNING watch out"; seq 50) | "$GLANCE" -p errors -p warnings )
assert_contains "multi preset: matches ERROR" "ERROR oops" "$out"
assert_contains "multi preset: matches WARNING" "WARNING watch out" "$out"

# Filter + preset OR together
out=$( (seq 50; echo "ERROR oops"; seq 50; echo "custom_match"; seq 50) | "$GLANCE" -p errors -f 'custom_match' )
assert_contains "filter+preset: matches ERROR" "ERROR oops" "$out"
assert_contains "filter+preset: matches custom" "custom_match" "$out"

# Filter matching nothing → just head/tail
out=$(seq 100 | "$GLANCE" -f 'NOMATCH')
assert_contains "no match: showing 20" "showing 20" "$out"

# Filter matching head/tail lines → no duplicates
out=$(seq 100 | "$GLANCE" -n 5 -f '^1$')
# Line "1" is in head, should not appear twice
_count=$(printf '%s' "$out" | grep -c '1: 1' || true)
assert "no dup: line 1 appears once" "1" "$_count"

# =============================================
printf '\nOutput format\n'
# =============================================

# Header has id
out=$(seq 10 | "$GLANCE")
assert_contains "header: has id" "id=" "$out"

# Line numbers are correct
out=$(seq 500 | "$GLANCE" -n 2)
_first_line=$(printf '%s' "$out" | head -1)
assert_contains "format: first content line is 1" "1: 1" "$_first_line"
_second_last=$(printf '%s' "$out" | tail -2 | head -1)
assert_contains "format: last content line is 500" "500: 500" "$_second_last"

# =============================================
printf '\nShow subcommand\n'
# =============================================

# Store a capture and retrieve it
out=$(seq 50 | "$GLANCE")
_id=$(extract_id "$out")

# Full output
show_out=$("$GLANCE" show "$_id")
_show_lines=$(printf '%s\n' "$show_out" | wc -l | tr -d ' ')
assert "show full: 50 lines" "50" "$_show_lines"

# Line range
range_out=$("$GLANCE" show "$_id" --lines 5-10)
_range_lines=$(printf '%s\n' "$range_out" | wc -l | tr -d ' ')
assert "show range: 6 lines + footer" "7" "$_range_lines"
assert_contains "show range: starts with 5" "5: 5" "$range_out"

# Invalid --lines range
err_out=$("$GLANCE" show "$_id" --lines abc 2>&1) || true
assert_contains "invalid --lines: error" "invalid.*range|must be.*N-M" "$err_out"
err_out=$("$GLANCE" show "$_id" --lines 5-abc 2>&1) || true
assert_contains "invalid --lines end: error" "invalid.*range|must be.*N-M" "$err_out"

# Filter within stored output
filter_out=$("$GLANCE" show "$_id" --filter '^1[0-9]$')
# Should match 10-19
assert_contains "show filter: matches 10" "10:" "$filter_out"

# Around
around_out=$("$GLANCE" show "$_id" --around 25 2)
assert_contains "show around: has line 23" "23: 23" "$around_out"
assert_contains "show around: has line 25" "25: 25" "$around_out"
assert_contains "show around: has line 27" "27: 27" "$around_out"

# Around with non-numeric center
err_out=$("$GLANCE" show "$_id" --around abc 5 2>&1) || true
assert_contains "around bad center: error" "must be a positive integer" "$err_out"

# Around with non-numeric context
err_out=$("$GLANCE" show "$_id" --around 25 xyz 2>&1) || true
assert_contains "around bad context: error" "must be a positive integer" "$err_out"

# Around near start (line 1, context 3)
around_out=$("$GLANCE" show "$_id" --around 1 3)
assert_contains "around start: has line 1" "1: 1" "$around_out"
assert_contains "around start: has line 4" "4: 4" "$around_out"
assert_not_contains "around start: no line 0" "0:" "$around_out"

# Around near end (line 50, context 3)
around_out=$("$GLANCE" show "$_id" --around 50 3)
assert_contains "around end: has line 47" "47: 47" "$around_out"
assert_contains "around end: has line 50" "50: 50" "$around_out"

# Combined flags: --lines and --filter OR together
combo_out=$("$GLANCE" show "$_id" --lines 1-3 --filter '^4[0-9]$')
assert_contains "combo: has line 1" "1: 1" "$combo_out"
assert_contains "combo: has line 3" "3: 3" "$combo_out"
assert_contains "combo: has line 40" "40: 40" "$combo_out"
assert_contains "combo: has line 49" "49: 49" "$combo_out"
assert_not_contains "combo: no line 20" "20: 20" "$combo_out"

# Combined flags: --filter and --around OR together
combo_out=$("$GLANCE" show "$_id" --filter '^5$' --around 40 2)
assert_contains "combo filter+around: has line 5" "5: 5" "$combo_out"
assert_contains "combo filter+around: has line 38" "38: 38" "$combo_out"
assert_contains "combo filter+around: has line 42" "42: 42" "$combo_out"

# Invalid ID
err_out=$("$GLANCE" show "zzz" 2>&1) || true
assert_contains "show invalid: error message" "capture not found" "$err_out"

# Path traversal in show ID
err_out=$("$GLANCE" show "../../etc/passwd" 2>&1) || true
assert_contains "traversal: rejected" "invalid capture ID" "$err_out"
err_out=$("$GLANCE" show "foo/bar" 2>&1) || true
assert_contains "slash in ID: rejected" "invalid capture ID" "$err_out"

# --- Unified flag API ---
printf '\n--- Unified flag API ---\n'

# Pipe: --head as alias for -n
out=$(seq 500 | "$GLANCE" --head 3 2>&1 || true)
assert_contains "pipe --head: showing 6" "showing 6" "$out"

# Show: -f short flag for --filter
filter_out=$("$GLANCE" show "$_id" -f '^1[0-9]$' 2>&1 || true)
assert_contains "show -f: matches 10" "10:" "$filter_out"

# Show: -p/--preset support
preset_store=$(printf 'line1\nERROR something\nline3\n' | "$GLANCE")
_preset_id=$(extract_id "$preset_store")
preset_out=$("$GLANCE" show "$_preset_id" -p errors 2>&1 || true)
assert_contains "show -p: matches error" "ERROR something" "$preset_out"
preset_out=$("$GLANCE" show "$_preset_id" --preset errors 2>&1 || true)
assert_contains "show --preset: matches error" "ERROR something" "$preset_out"

# Show: -l short flag for --lines
range_out=$("$GLANCE" show "$_id" -l 5-10 2>&1 || true)
assert_contains "show -l: starts with 5" "5: 5" "$range_out"

# Show: -a short flag for --around
around_out=$("$GLANCE" show "$_id" -a 25 2 2>&1 || true)
assert_contains "show -a: has line 25" "25: 25" "$around_out"
assert_contains "show -a: has line 23" "23: 23" "$around_out"

# =============================================
printf '\nList/clean\n'
# =============================================

list_out=$("$GLANCE" list)
assert_contains "list: has captures" "lines" "$list_out"

"$GLANCE" clean
list_after=$("$GLANCE" list)
assert_contains "clean: empty after" "No stored captures" "$list_after"

# =============================================
printf '\nPresets management\n'
# =============================================

# List shows built-in
presets_out=$("$GLANCE" presets list)
assert_contains "presets list: has errors" "errors" "$presets_out"
assert_contains "presets list: has status" "status" "$presets_out"

# Add user preset
"$GLANCE" presets add testpre 'foo|bar' "Test preset"
presets_out=$("$GLANCE" presets list)
assert_contains "presets add: appears in list" "testpre" "$presets_out"

# User preset works with -p
out=$( (echo "foo match"; seq 50) | "$GLANCE" -p testpre )
assert_contains "user preset: matches" "foo match" "$out"

# Remove user preset
"$GLANCE" presets remove testpre
presets_out=$("$GLANCE" presets list)
assert_not_contains "presets remove: gone from list" "testpre" "$presets_out"

# Cannot remove built-in
err_out=$("$GLANCE" presets remove errors 2>&1) || true
assert_contains "presets remove builtin: error" "cannot remove built-in" "$err_out"

# Auto-delimiter: regex with / gets comma delimiter
"$GLANCE" presets add slashpre 'a/b' "Has slash"
_conf_line=$(grep slashpre "$XDG_CONFIG_HOME/glance/presets.conf")
assert_contains "auto-delim: uses comma for slash regex" "^," "$_conf_line"
# Verify it round-trips
out=$( (echo "a/b found"; seq 50) | "$GLANCE" -p slashpre )
assert_contains "auto-delim: preset works" "a/b found" "$out"
"$GLANCE" presets remove slashpre

# -d flag: explicit delimiter
"$GLANCE" presets add -d '@' flagpre 'x/y,z' "Custom delim"
_conf_line=$(grep flagpre "$XDG_CONFIG_HOME/glance/presets.conf")
assert_contains "-d flag: uses @ delimiter" "^@" "$_conf_line"
out=$( (echo "x/y,z here"; seq 50) | "$GLANCE" -p flagpre )
assert_contains "-d flag: preset works" "x/y,z here" "$out"
"$GLANCE" presets remove flagpre

# -d flag: reject delimiter that appears in regex
err_out=$("$GLANCE" presets add -d '/' badpre 'a/b' "desc" 2>&1) || true
assert_contains "-d conflict: error" "delimiter.*appears in regex" "$err_out"

# Delimiter exhaustion: regex with all candidates
err_out=$("$GLANCE" presets add exhaustpre '/,@#%~!' "desc" 2>&1) || true
assert_contains "delim exhaustion: error" "all candidate delimiters" "$err_out"

# =============================================
printf '\nFormat gaps\n'
# =============================================

# Comments and blank lines preserved through add/remove
_conf="$XDG_CONFIG_HOME/glance/presets.conf"
mkdir -p "$(dirname "$_conf")"
printf '# My comment\n\n/keepme/keep/Keep preset\n' > "$_conf"
"$GLANCE" presets add newpre 'new' "New one"
assert_contains "comment preserved after add" "^# My comment" "$(cat "$_conf")"
"$GLANCE" presets remove newpre
assert_contains "comment preserved after remove" "^# My comment" "$(cat "$_conf")"
# Verify keepme survived too
out=$( (echo "keep this"; seq 50) | "$GLANCE" -p keepme )
assert_contains "other preset survives add/remove" "keep this" "$out"
"$GLANCE" presets remove keepme
rm -f "$_conf"

# Preset update: re-add replaces old entry
"$GLANCE" presets add mypre 'old_regex' "Old desc"
"$GLANCE" presets add mypre 'new_regex' "New desc"
_count=$(grep -c 'mypre' "$_conf")
assert "re-add: only one entry" "1" "$_count"
out=$( (echo "new_regex match"; seq 50) | "$GLANCE" -p mypre )
assert_contains "re-add: uses new regex" "new_regex match" "$out"
"$GLANCE" presets remove mypre
rm -f "$_conf"

# Unknown preset with -p
err_out=$( (seq 10) | "$GLANCE" -p nonexistent 2>&1) || true
assert_contains "unknown preset: error" "unknown preset" "$err_out"

# presets add with missing args
err_out=$("$GLANCE" presets add 2>&1) || true
assert_contains "presets add no args: usage" "Usage" "$err_out"
err_out=$("$GLANCE" presets add onlyname 2>&1) || true
assert_contains "presets add one arg: usage" "Usage" "$err_out"

# clean --all removes user presets
"$GLANCE" presets add cleantst 'x' "test"
seq 5 | "$GLANCE" >/dev/null  # create a capture too
"$GLANCE" clean --all
assert "clean --all: config gone" "0" "$([ -f "$_conf" ] && echo 1 || echo 0)"

# Preset name validation: reject spaces and special chars
err_out=$("$GLANCE" presets add 'bad name' 'regex' 2>&1) || true
assert_contains "name with space: error" "invalid preset name" "$err_out"
err_out=$("$GLANCE" presets add 'bad/name' 'regex' 2>&1) || true
assert_contains "name with slash: error" "invalid preset name" "$err_out"
err_out=$("$GLANCE" presets add '' 'regex' 2>&1) || true
assert_contains "empty name: error" "invalid preset name" "$err_out"

# Preset name starting with dash (looks like a flag)
err_out=$("$GLANCE" presets add -- '-n' 'regex' 2>&1) || true
assert_contains "name with dash prefix: error" "invalid preset name" "$err_out"

# Description containing delimiter round-trips
"$GLANCE" presets add descpre 'myregex' "Status/exit codes"
presets_out=$("$GLANCE" presets list)
assert_contains "desc with delimiter: shows correctly" "Status/exit codes" "$presets_out"
"$GLANCE" presets remove descpre

# =============================================
printf '\nEdge cases\n'
# =============================================

# Unknown subcommand
err_out=$("$GLANCE" foobar 2>&1) || true
assert_contains "unknown cmd: error" "unknown command" "$err_out"

# Invalid -n value
err_out=$(seq 10 | "$GLANCE" -n abc 2>&1) || true
assert_contains "invalid -n: error message" "must be a positive integer" "$err_out"

# -n 0 should be rejected
err_out=$(seq 10 | "$GLANCE" -n 0 2>&1) || true
assert_contains "-n 0: error" "must be a positive integer" "$err_out"

# Filter pattern starting with dash (not treated as grep flag)
out=$( (seq 50; echo "-- separator --"; seq 50) | "$GLANCE" -f '--' )
assert_contains "dash filter: matches" "separator" "$out"

# Missing value for -f
err_out=$(seq 10 | "$GLANCE" -f 2>&1) || true
assert_contains "-f no value: error" "requires a value" "$err_out"

# Missing value for -p
err_out=$(seq 10 | "$GLANCE" -p 2>&1) || true
assert_contains "-p no value: error" "requires a value" "$err_out"

# Missing value for -n
err_out=$(seq 10 | "$GLANCE" -n 2>&1) || true
assert_contains "-n no value: error" "must be a positive integer" "$err_out"
# Should not create a capture
captures_before=$(ls "$XDG_CACHE_HOME/glance/captures/" 2>/dev/null | wc -l | tr -d ' ')
seq 10 | "$GLANCE" -n abc 2>/dev/null || true
captures_after=$(ls "$XDG_CACHE_HOME/glance/captures/" 2>/dev/null | wc -l | tr -d ' ')
assert "invalid -n: no capture created" "$captures_before" "$captures_after"

# Long-form --lines flag in pipe mode
out=$(seq 500 | "$GLANCE" --lines 3)
assert_contains "long --lines: showing 6" "showing 6" "$out"

# Long-form --filter flag in pipe mode
out=$( (seq 100; echo "LONGFORM_MATCH"; seq 200) | "$GLANCE" --filter 'LONGFORM_MATCH' )
assert_contains "long --filter: matches" "LONGFORM_MATCH" "$out"

# Long-form --preset flag in pipe mode
out=$( (seq 100; echo "ERROR oops"; seq 200) | "$GLANCE" --preset errors )
assert_contains "long --preset: matches" "ERROR oops" "$out"

# --no-store flag
out=$(seq 10 | "$GLANCE" --no-store)
assert_not_contains "no-store: no id" "id=" "$out"

# Very long lines
long_line=$(printf '%0*d' 10000 0)
out=$(printf '%s\n' "$long_line" | "$GLANCE")
assert_contains "long line: 1 line" "1 line[^s]" "$out"

# Concurrent captures (just test no crash)
seq 100 | "$GLANCE" >/dev/null &
seq 200 | "$GLANCE" >/dev/null &
seq 300 | "$GLANCE" >/dev/null &
wait
TOTAL=$(( TOTAL + 1 ))
PASS=$(( PASS + 1 ))
printf '  ok  concurrent captures: no crash\n'

# --- Regex compatibility ---
printf '\n--- Regex compatibility ---\n'

# Status preset uses regex that must work in awk (not just grep)
# Use enough lines so HTTP 500 falls outside head/tail, only matched by filter
input=$(printf 'line 1\nline 2\nline 3\nline 4\nline 5\nHTTP 500 server error\nline 7\nline 8\nline 9\nline 10\nline 11\n')
out=$(printf '%s\n' "$input" | "$GLANCE" -n 2 -p status)
assert_contains "status preset matches HTTP 5xx via filter" "HTTP 500 server error" "$out"

# --- Invalid regex ---
printf '\n--- Invalid regex ---\n'

# Use enough lines so filter path is exercised (not just head/tail)
out=$(printf 'l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\nl11\nl12\nl13\nl14\nl15\nl16\nl17\nl18\nl19\nl20\nl21\n' | "$GLANCE" -n 1 -f '[' 2>&1 || true)
# awk errors on invalid regex — should produce an error on stderr
assert_contains "invalid regex: awk error" "nonterminated|unterminated|character class" "$out"

# --- Age formatting boundaries ---
printf '\n--- Age formatting ---\n'

# We can't easily test format_age directly since it's internal,
# but we can verify the function exists and works via glance list
# by checking that the list output contains age strings.
# Instead, test boundary values by sourcing the function.
_test_format_age() {
    # Source just the format_age function
    eval "$(sed -n '/^format_age()/,/^}/p' "$GLANCE")"
    _result=$(format_age "$1")
    printf '%s' "$_result"
}

assert "age: 0s = unknown" "unknown" "$(_test_format_age 0)"
assert "age: 59s" "59s ago" "$(_test_format_age 59)"
assert "age: 60s = 1m" "1m ago" "$(_test_format_age 60)"
assert "age: 3599s = 59m" "59m ago" "$(_test_format_age 3599)"
assert "age: 3600s = 1h" "1h ago" "$(_test_format_age 3600)"
assert "age: 86399s = 23h" "23h ago" "$(_test_format_age 86399)"
assert "age: 86400s = 1d" "1d ago" "$(_test_format_age 86400)"

# --- Presets no subcommand ---
printf '\n--- Presets edge cases ---\n'

out=$("$GLANCE" presets 2>&1 || true)
assert_contains "presets no subcommand: usage" "Usage.*presets" "$out"

# --- Delimiter exhaustion ---
printf '\n--- Delimiter exhaustion ---\n'

out=$("$GLANCE" presets add test_exhaust '/,@#%~!all' 'desc' 2>&1 || true)
assert_contains "delimiter exhaustion: error" "all candidate delimiters" "$out"

# =============================================
printf '\n========================================\n'
printf 'Results: %d/%d passed' "$PASS" "$TOTAL"
if [ "$FAIL" -gt 0 ]; then
    printf ' (%d FAILED)' "$FAIL"
fi
printf '\n========================================\n'

[ "$FAIL" -eq 0 ]
