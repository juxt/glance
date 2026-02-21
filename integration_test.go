package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
)

var glanceBin string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "glance-test-bin-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	bin := filepath.Join(tmp, "glance")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build: %s\n%s\n", err, out)
		os.Exit(1)
	}
	glanceBin = bin

	os.Exit(m.Run())
}

// run executes glance with stdin and args, returning stdout, stderr, and exit code.
func run(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(glanceBin, args...)
	cmd.Stdin = strings.NewReader(stdin)
	// Each test gets isolated cache/config
	cacheDir := t.TempDir()
	configDir := t.TempDir()
	cmd.Env = append(os.Environ(),
		"XDG_CACHE_HOME="+cacheDir,
		"XDG_CONFIG_HOME="+configDir,
	)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec error: %v", err)
		}
	}
	return stdout.String(), stderr.String(), exitCode
}

// runWithDirs is like run but lets caller control cache/config dirs across calls.
type testEnv struct {
	t         *testing.T
	cacheDir  string
	configDir string
}

func newTestEnv(t *testing.T) *testEnv {
	return &testEnv{
		t:         t,
		cacheDir:  t.TempDir(),
		configDir: t.TempDir(),
	}
}

func (e *testEnv) run(stdin string, args ...string) (string, string, int) {
	e.t.Helper()
	cmd := exec.Command(glanceBin, args...)
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Env = append(os.Environ(),
		"XDG_CACHE_HOME="+e.cacheDir,
		"XDG_CONFIG_HOME="+e.configDir,
	)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			e.t.Fatalf("exec error: %v", err)
		}
	}
	return stdout.String(), stderr.String(), exitCode
}

func extractID(output string) string {
	re := regexp.MustCompile(`id=(\S+)`)
	m := re.FindStringSubmatch(output)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func assertContains(t *testing.T, desc, output, pattern string) {
	t.Helper()
	matched, err := regexp.MatchString(pattern, output)
	if err != nil {
		t.Errorf("%s: bad regex %q: %v", desc, pattern, err)
		return
	}
	if !matched {
		t.Errorf("%s: output does not match %q\noutput:\n%s", desc, pattern, truncate(output, 500))
	}
}

func assertNotContains(t *testing.T, desc, output, pattern string) {
	t.Helper()
	matched, _ := regexp.MatchString(pattern, output)
	if matched {
		t.Errorf("%s: output should not match %q\noutput:\n%s", desc, pattern, truncate(output, 500))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}

func seqInput(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "%d\n", i)
	}
	return b.String()
}

// ===== Integration Tests =====

func TestPipe(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		out, _, _ := run(t, "")
		assertContains(t, "0 lines", out, `0 lines`)
		assertContains(t, "showing 0", out, `showing 0`)
	})

	t.Run("single line", func(t *testing.T) {
		out, _, _ := run(t, "hello\n")
		assertContains(t, "singular", out, `1 line[^s]`)
		assertContains(t, "content", out, `hello`)
	})

	t.Run("no trailing newline", func(t *testing.T) {
		out, _, _ := run(t, "hello")
		assertContains(t, "1 line", out, `1 line[^s]`)
		assertContains(t, "content", out, `hello`)
	})

	t.Run("short input", func(t *testing.T) {
		out, _, _ := run(t, seqInput(5))
		assertContains(t, "5 lines", out, `5 lines`)
		assertContains(t, "showing 5", out, `showing 5`)
		assertContains(t, "sections 1-5", out, `sections: 1-5`)
	})

	t.Run("exact boundary", func(t *testing.T) {
		out, _, _ := run(t, seqInput(20), "-n", "10")
		assertContains(t, "20 lines", out, `20 lines`)
		assertContains(t, "showing 20", out, `showing 20`)
	})

	t.Run("large input", func(t *testing.T) {
		out, _, _ := run(t, seqInput(1000))
		assertContains(t, "1000 lines", out, `1000 lines`)
		assertContains(t, "showing 20", out, `showing 20`)
		assertContains(t, "has line 1", out, `\b1: 1\b`)
		assertContains(t, "has line 1000", out, `1000: 1000`)
	})

	t.Run("custom n", func(t *testing.T) {
		out, _, _ := run(t, seqInput(500), "-n", "3")
		assertContains(t, "showing 6", out, `showing 6`)
		assertContains(t, "sections", out, `sections: 1-3, 498-500`)
	})

	t.Run("binary input", func(t *testing.T) {
		// Should not crash
		run(t, "line1\nline2\x00binary\nline3\n")
	})
}

func TestFiltering(t *testing.T) {
	t.Run("filter matches middle", func(t *testing.T) {
		input := seqInput(100) + "MATCH here\n" + seqInput(200)
		out, _, _ := run(t, input, "-f", "MATCH")
		assertContains(t, "matches middle", out, `101: MATCH here`)
	})

	t.Run("preset errors", func(t *testing.T) {
		input := seqInput(100) + "ERROR bad thing\n" + seqInput(200)
		out, _, _ := run(t, input, "-p", "errors")
		assertContains(t, "matches ERROR", out, `ERROR bad thing`)
	})

	t.Run("multi preset", func(t *testing.T) {
		input := seqInput(50) + "ERROR oops\n" + seqInput(50) + "WARNING watch out\n" + seqInput(50)
		out, _, _ := run(t, input, "-p", "errors", "-p", "warnings")
		assertContains(t, "matches ERROR", out, `ERROR oops`)
		assertContains(t, "matches WARNING", out, `WARNING watch out`)
	})

	t.Run("filter plus preset", func(t *testing.T) {
		input := seqInput(50) + "ERROR oops\n" + seqInput(50) + "custom_match\n" + seqInput(50)
		out, _, _ := run(t, input, "-p", "errors", "-f", "custom_match")
		assertContains(t, "matches ERROR", out, `ERROR oops`)
		assertContains(t, "matches custom", out, `custom_match`)
	})

	t.Run("no match", func(t *testing.T) {
		out, _, _ := run(t, seqInput(100), "-f", "NOMATCH")
		assertContains(t, "showing 20", out, `showing 20`)
	})

	t.Run("no duplicate", func(t *testing.T) {
		out, _, _ := run(t, seqInput(100), "-n", "5", "-f", `^1$`)
		// Line "1" is in head, should not appear twice
		count := strings.Count(out, "1: 1\n")
		if count != 1 {
			t.Errorf("no dup: line 1 appears %d times, want 1", count)
		}
	})
}

func TestOutputFormat(t *testing.T) {
	t.Run("has id", func(t *testing.T) {
		out, _, _ := run(t, seqInput(10))
		assertContains(t, "has id", out, `id=`)
	})

	t.Run("line numbers correct", func(t *testing.T) {
		out, _, _ := run(t, seqInput(500), "-n", "2")
		lines := strings.Split(strings.TrimSpace(out), "\n")
		assertContains(t, "first line is 1", lines[0], `1: 1`)
		// Second to last line (last is footer)
		assertContains(t, "last content line is 500", lines[len(lines)-2], `500: 500`)
	})
}

func TestShow(t *testing.T) {
	env := newTestEnv(t)

	// Store a capture
	out, _, _ := env.run(seqInput(50))
	id := extractID(out)
	if id == "" {
		t.Fatal("failed to extract ID from pipe output")
	}

	t.Run("full output", func(t *testing.T) {
		out, _, _ := env.run("", "show", id)
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		if len(lines) != 50 {
			t.Errorf("show full: got %d lines, want 50", len(lines))
		}
	})

	t.Run("line range", func(t *testing.T) {
		out, _, _ := env.run("", "show", id, "--lines", "5-10")
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		// 6 content lines + 1 footer
		if len(lines) != 7 {
			t.Errorf("show range: got %d lines, want 7", len(lines))
		}
		assertContains(t, "starts with 5", out, `5: 5`)
	})

	t.Run("invalid lines range", func(t *testing.T) {
		_, stderr, _ := env.run("", "show", id, "--lines", "abc")
		assertContains(t, "error", stderr, `invalid.*range|must be.*N-M`)
	})

	t.Run("invalid lines end", func(t *testing.T) {
		_, stderr, _ := env.run("", "show", id, "--lines", "5-abc")
		assertContains(t, "error", stderr, `invalid.*range|must be.*N-M`)
	})

	t.Run("filter within stored", func(t *testing.T) {
		out, _, _ := env.run("", "show", id, "--filter", `^1[0-9]$`)
		assertContains(t, "matches 10", out, `10:`)
	})

	t.Run("around", func(t *testing.T) {
		out, _, _ := env.run("", "show", id, "--around", "25", "2")
		assertContains(t, "has line 23", out, `23: 23`)
		assertContains(t, "has line 25", out, `25: 25`)
		assertContains(t, "has line 27", out, `27: 27`)
	})

	t.Run("around bad center", func(t *testing.T) {
		_, stderr, _ := env.run("", "show", id, "--around", "abc", "5")
		assertContains(t, "error", stderr, `must be a positive integer`)
	})

	t.Run("around bad context", func(t *testing.T) {
		_, stderr, _ := env.run("", "show", id, "--around", "25", "xyz")
		assertContains(t, "error", stderr, `must be a positive integer`)
	})

	t.Run("around near start", func(t *testing.T) {
		out, _, _ := env.run("", "show", id, "--around", "1", "3")
		assertContains(t, "has line 1", out, `1: 1`)
		assertContains(t, "has line 4", out, `4: 4`)
		assertNotContains(t, "no line 0", out, `0:`)
	})

	t.Run("around near end", func(t *testing.T) {
		out, _, _ := env.run("", "show", id, "--around", "50", "3")
		assertContains(t, "has line 47", out, `47: 47`)
		assertContains(t, "has line 50", out, `50: 50`)
	})

	t.Run("combo lines and filter", func(t *testing.T) {
		out, _, _ := env.run("", "show", id, "--lines", "1-3", "--filter", `^4[0-9]$`)
		assertContains(t, "has line 1", out, `1: 1`)
		assertContains(t, "has line 3", out, `3: 3`)
		assertContains(t, "has line 40", out, `40: 40`)
		assertContains(t, "has line 49", out, `49: 49`)
		assertNotContains(t, "no line 20", out, `20: 20`)
	})

	t.Run("combo filter and around", func(t *testing.T) {
		out, _, _ := env.run("", "show", id, "--filter", `^5$`, "--around", "40", "2")
		assertContains(t, "has line 5", out, `5: 5`)
		assertContains(t, "has line 38", out, `38: 38`)
		assertContains(t, "has line 42", out, `42: 42`)
	})

	t.Run("invalid id", func(t *testing.T) {
		_, stderr, _ := env.run("", "show", "zzz")
		assertContains(t, "error", stderr, `capture not found`)
	})

	t.Run("path traversal", func(t *testing.T) {
		_, stderr, _ := env.run("", "show", "../../etc/passwd")
		assertContains(t, "rejected", stderr, `invalid capture ID`)

		_, stderr, _ = env.run("", "show", "foo/bar")
		assertContains(t, "slash rejected", stderr, `invalid capture ID`)
	})
}

func TestUnifiedFlags(t *testing.T) {
	env := newTestEnv(t)

	t.Run("pipe --head", func(t *testing.T) {
		out, _, _ := env.run(seqInput(500), "--head", "3")
		assertContains(t, "showing 6", out, `showing 6`)
	})

	// Store a capture for show flag tests
	out, _, _ := env.run(seqInput(50))
	id := extractID(out)

	t.Run("show -f", func(t *testing.T) {
		out, _, _ := env.run("", "show", id, "-f", `^1[0-9]$`)
		assertContains(t, "matches 10", out, `10:`)
	})

	t.Run("show -p and --preset", func(t *testing.T) {
		storeOut, _, _ := env.run("line1\nERROR something\nline3\n")
		pid := extractID(storeOut)

		out, _, _ := env.run("", "show", pid, "-p", "errors")
		assertContains(t, "-p matches", out, `ERROR something`)

		out, _, _ = env.run("", "show", pid, "--preset", "errors")
		assertContains(t, "--preset matches", out, `ERROR something`)
	})

	t.Run("show -l", func(t *testing.T) {
		out, _, _ := env.run("", "show", id, "-l", "5-10")
		assertContains(t, "starts with 5", out, `5: 5`)
	})

	t.Run("show -a", func(t *testing.T) {
		out, _, _ := env.run("", "show", id, "-a", "25", "2")
		assertContains(t, "has line 25", out, `25: 25`)
		assertContains(t, "has line 23", out, `23: 23`)
	})
}

func TestListClean(t *testing.T) {
	env := newTestEnv(t)

	// Create some captures
	env.run(seqInput(10))
	env.run(seqInput(20))

	t.Run("list has captures", func(t *testing.T) {
		out, _, _ := env.run("", "list")
		assertContains(t, "has captures", out, `lines`)
	})

	t.Run("clean empties", func(t *testing.T) {
		env.run("", "clean")
		out, _, _ := env.run("", "list")
		assertContains(t, "empty after", out, `No stored captures`)
	})

	t.Run("clean --all", func(t *testing.T) {
		// Create capture + preset
		env.run(seqInput(5))
		env.run("", "presets", "add", "cleantst", "x", "test")

		env.run("", "clean", "--all")

		confPath := filepath.Join(env.configDir, "glance", "presets.conf")
		if _, err := os.Stat(confPath); err == nil {
			t.Error("clean --all: config file should be gone")
		}
	})
}

func TestPresets(t *testing.T) {
	env := newTestEnv(t)

	t.Run("list shows builtin", func(t *testing.T) {
		out, _, _ := env.run("", "presets", "list")
		assertContains(t, "has errors", out, `errors`)
		assertContains(t, "has status", out, `status`)
	})

	t.Run("add and use", func(t *testing.T) {
		env.run("", "presets", "add", "testpre", "foo|bar", "Test preset")

		out, _, _ := env.run("", "presets", "list")
		assertContains(t, "appears in list", out, `testpre`)

		out, _, _ = env.run("foo match\n"+seqInput(50), "-p", "testpre")
		assertContains(t, "matches", out, `foo match`)
	})

	t.Run("remove", func(t *testing.T) {
		env.run("", "presets", "remove", "testpre")
		out, _, _ := env.run("", "presets", "list")
		assertNotContains(t, "gone from list", out, `testpre`)
	})

	t.Run("cannot remove builtin", func(t *testing.T) {
		_, stderr, _ := env.run("", "presets", "remove", "errors")
		assertContains(t, "error", stderr, `cannot remove built-in`)
	})

	t.Run("auto delimiter", func(t *testing.T) {
		env.run("", "presets", "add", "slashpre", "a/b", "Has slash")

		confPath := filepath.Join(env.configDir, "glance", "presets.conf")
		data, _ := os.ReadFile(confPath)
		assertContains(t, "uses comma", string(data), `(?m)^,`)

		// Round-trips
		out, _, _ := env.run("a/b found\n"+seqInput(50), "-p", "slashpre")
		assertContains(t, "preset works", out, `a/b found`)

		env.run("", "presets", "remove", "slashpre")
	})

	t.Run("-d flag", func(t *testing.T) {
		env.run("", "presets", "add", "-d", "@", "flagpre", "x/y,z", "Custom delim")

		confPath := filepath.Join(env.configDir, "glance", "presets.conf")
		data, _ := os.ReadFile(confPath)
		assertContains(t, "uses @", string(data), `(?m)^@`)

		out, _, _ := env.run("x/y,z here\n"+seqInput(50), "-p", "flagpre")
		assertContains(t, "preset works", out, `x/y,z here`)

		env.run("", "presets", "remove", "flagpre")
	})

	t.Run("-d conflict", func(t *testing.T) {
		_, stderr, _ := env.run("", "presets", "add", "-d", "/", "badpre", "a/b", "desc")
		assertContains(t, "error", stderr, `delimiter.*appears in regex`)
	})

	t.Run("delimiter exhaustion", func(t *testing.T) {
		_, stderr, _ := env.run("", "presets", "add", "exhaustpre", "/,@#%~!", "desc")
		assertContains(t, "error", stderr, `all candidate delimiters`)
	})

	t.Run("comments preserved", func(t *testing.T) {
		// Fresh env for this test
		env2 := newTestEnv(t)
		confDir := filepath.Join(env2.configDir, "glance")
		os.MkdirAll(confDir, 0o755)
		confPath := filepath.Join(confDir, "presets.conf")
		os.WriteFile(confPath, []byte("# My comment\n\n/keepme/keep/Keep preset\n"), 0o644)

		env2.run("", "presets", "add", "newpre", "new", "New one")
		data, _ := os.ReadFile(confPath)
		assertContains(t, "comment after add", string(data), `(?m)^# My comment`)

		env2.run("", "presets", "remove", "newpre")
		data, _ = os.ReadFile(confPath)
		assertContains(t, "comment after remove", string(data), `(?m)^# My comment`)

		// keepme survived
		out, _, _ := env2.run("keep this\n"+seqInput(50), "-p", "keepme")
		assertContains(t, "other preset survives", out, `keep this`)
	})

	t.Run("re-add replaces", func(t *testing.T) {
		env2 := newTestEnv(t)
		env2.run("", "presets", "add", "mypre", "old_regex", "Old desc")
		env2.run("", "presets", "add", "mypre", "new_regex", "New desc")

		confPath := filepath.Join(env2.configDir, "glance", "presets.conf")
		data, _ := os.ReadFile(confPath)
		count := strings.Count(string(data), "mypre")
		if count != 1 {
			t.Errorf("re-add: got %d entries, want 1", count)
		}

		out, _, _ := env2.run("new_regex match\n"+seqInput(50), "-p", "mypre")
		assertContains(t, "uses new regex", out, `new_regex match`)
	})

	t.Run("unknown preset", func(t *testing.T) {
		_, stderr, _ := env.run(seqInput(10), "-p", "nonexistent")
		assertContains(t, "error", stderr, `unknown preset`)
	})

	t.Run("missing args", func(t *testing.T) {
		_, stderr, _ := env.run("", "presets", "add")
		assertContains(t, "no args", stderr, `Usage`)

		_, stderr, _ = env.run("", "presets", "add", "onlyname")
		assertContains(t, "one arg", stderr, `Usage`)
	})

	t.Run("name validation", func(t *testing.T) {
		_, stderr, _ := env.run("", "presets", "add", "bad name", "regex")
		assertContains(t, "space", stderr, `invalid preset name`)

		_, stderr, _ = env.run("", "presets", "add", "bad/name", "regex")
		assertContains(t, "slash", stderr, `invalid preset name`)

		_, stderr, _ = env.run("", "presets", "add", "", "regex")
		assertContains(t, "empty", stderr, `invalid preset name`)

		_, stderr, _ = env.run("", "presets", "add", "--", "-n", "regex")
		assertContains(t, "dash prefix", stderr, `invalid preset name`)
	})

	t.Run("desc with delimiter", func(t *testing.T) {
		env2 := newTestEnv(t)
		env2.run("", "presets", "add", "descpre", "myregex", "Status/exit codes")
		out, _, _ := env2.run("", "presets", "list")
		assertContains(t, "shows correctly", out, `Status/exit codes`)
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("unknown command", func(t *testing.T) {
		_, stderr, _ := run(t, "", "foobar")
		assertContains(t, "error", stderr, `unknown command`)
	})

	t.Run("invalid -n", func(t *testing.T) {
		_, stderr, _ := run(t, seqInput(10), "-n", "abc")
		assertContains(t, "error", stderr, `must be a positive integer`)
	})

	t.Run("-n 0", func(t *testing.T) {
		_, stderr, _ := run(t, seqInput(10), "-n", "0")
		assertContains(t, "error", stderr, `must be a positive integer`)
	})

	t.Run("dash filter", func(t *testing.T) {
		input := seqInput(50) + "-- separator --\n" + seqInput(50)
		out, _, _ := run(t, input, "-f", "--")
		assertContains(t, "matches", out, `separator`)
	})

	t.Run("missing -f value", func(t *testing.T) {
		_, stderr, _ := run(t, seqInput(10), "-f")
		assertContains(t, "error", stderr, `requires a value`)
	})

	t.Run("missing -p value", func(t *testing.T) {
		_, stderr, _ := run(t, seqInput(10), "-p")
		assertContains(t, "error", stderr, `requires a value`)
	})

	t.Run("missing -n value", func(t *testing.T) {
		_, stderr, _ := run(t, seqInput(10), "-n")
		assertContains(t, "error", stderr, `must be a positive integer`)
	})

	t.Run("invalid -n no capture", func(t *testing.T) {
		env := newTestEnv(t)
		capturesDir := filepath.Join(env.cacheDir, "glance", "captures")

		before := countFiles(capturesDir)
		env.run(seqInput(10), "-n", "abc")
		after := countFiles(capturesDir)
		if before != after {
			t.Errorf("invalid -n created capture: before=%d after=%d", before, after)
		}
	})

	t.Run("long --lines flag", func(t *testing.T) {
		out, _, _ := run(t, seqInput(500), "--lines", "3")
		assertContains(t, "showing 6", out, `showing 6`)
	})

	t.Run("long --filter flag", func(t *testing.T) {
		input := seqInput(100) + "LONGFORM_MATCH\n" + seqInput(200)
		out, _, _ := run(t, input, "--filter", "LONGFORM_MATCH")
		assertContains(t, "matches", out, `LONGFORM_MATCH`)
	})

	t.Run("long --preset flag", func(t *testing.T) {
		input := seqInput(100) + "ERROR oops\n" + seqInput(200)
		out, _, _ := run(t, input, "--preset", "errors")
		assertContains(t, "matches", out, `ERROR oops`)
	})

	t.Run("--no-store", func(t *testing.T) {
		out, _, _ := run(t, seqInput(10), "--no-store")
		assertNotContains(t, "no id", out, `id=`)
	})

	t.Run("long lines", func(t *testing.T) {
		longLine := strings.Repeat("0", 10000)
		out, _, _ := run(t, longLine+"\n")
		assertContains(t, "1 line", out, `1 line[^s]`)
	})

	t.Run("concurrent", func(t *testing.T) {
		env := newTestEnv(t)
		var wg sync.WaitGroup
		for _, n := range []int{100, 200, 300} {
			n := n
			wg.Add(1)
			go func() {
				defer wg.Done()
				env.run(seqInput(n))
			}()
		}
		wg.Wait()
		// If we get here without panic/deadlock, it passed
	})
}

func TestRegex(t *testing.T) {
	t.Run("status preset matches HTTP 5xx", func(t *testing.T) {
		var b strings.Builder
		for i := 1; i <= 11; i++ {
			if i == 6 {
				b.WriteString("HTTP 500 server error\n")
			} else {
				fmt.Fprintf(&b, "line %d\n", i)
			}
		}
		out, _, _ := run(t, b.String(), "-n", "2", "-p", "status")
		assertContains(t, "matches HTTP 500", out, `HTTP 500 server error`)
	})

	t.Run("invalid regex error", func(t *testing.T) {
		// 21 lines so filter path is exercised with -n 1
		var b strings.Builder
		for i := 1; i <= 21; i++ {
			fmt.Fprintf(&b, "l%d\n", i)
		}
		_, stderr, _ := run(t, b.String(), "-n", "1", "-f", "[")
		assertContains(t, "error", stderr, `nonterminated|unterminated|character class|error parsing regexp`)
	})
}

func TestPresetsEdgeCases(t *testing.T) {
	t.Run("no subcommand", func(t *testing.T) {
		_, stderr, _ := run(t, "", "presets")
		assertContains(t, "usage", stderr, `Usage.*presets`)
	})

	t.Run("delimiter exhaustion", func(t *testing.T) {
		_, stderr, _ := run(t, "", "presets", "add", "test_exhaust", "/,@#%~!all", "desc")
		assertContains(t, "error", stderr, `all candidate delimiters`)
	})
}

func countFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	return len(entries)
}
