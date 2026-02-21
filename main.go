package main

import (
	"fmt"
	"os"
)

var version = "dev"

func fatal(msg string) {
	fmt.Fprintf(os.Stderr, "glance: %s\n", msg)
	os.Exit(1)
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func main() {
	args := os.Args[1:]

	// No args and stdin is a terminal → error
	if len(args) == 0 && isTerminal() {
		fmt.Fprintf(os.Stderr, "glance: no input. Pipe command output to glance or use a subcommand.\n")
		fmt.Fprintf(os.Stderr, "Try: glance help\n")
		os.Exit(1)
	}

	if len(args) == 0 {
		doPipe(nil)
		return
	}

	switch args[0] {
	case "version", "--version", "-v":
		fmt.Println(version)
		return
	case "help":
		doHelp(args[1:])
	case "show":
		doShow(args[1:])
	case "list":
		doList()
	case "clean":
		doClean(args[1:])
	case "presets":
		doPresets(args[1:])
	default:
		if len(args[0]) > 0 && args[0][0] == '-' {
			// Pipe mode with flags
			doPipe(args)
		} else {
			fmt.Fprintf(os.Stderr, "glance: unknown command: %s\n", args[0])
			fmt.Fprintf(os.Stderr, "Try: glance help\n")
			os.Exit(1)
		}
	}
}

func doHelp(args []string) {
	if len(args) > 0 {
		switch args[0] {
		case "show":
			fmt.Print(`glance show — retrieve stored capture output

Usage:
  glance show <id>                    Full stored output
  glance show <id> -l 50-80           Lines 50 through 80
  glance show <id> -f regex           Filter within stored output
  glance show <id> -p errors          Filter with preset
  glance show <id> -a 247 5           5 lines context around line 247

Flags:
  -l, --lines N-M      Line range
  -f, --filter REGEX   Filter pattern (repeatable, OR)
  -p, --preset NAME    Preset filter (repeatable, OR)
  -a, --around N [C]   Context around line N (default C=5)

The <id> is the full ID shown in the glance footer when piping output.
Exact match required — use "glance list" to see all stored captures.
`)
			return
		case "list":
			fmt.Print(`glance list — show stored captures

Usage:
  glance list

Displays each capture's ID, line count, and age.
`)
			return
		case "clean":
			fmt.Print(`glance clean — purge stored captures

Usage:
  glance clean          Remove all stored captures
  glance clean --all    Also remove user presets
`)
			return
		case "presets":
			fmt.Print(`glance presets — manage filter presets

Usage:
  glance presets list                        Show all presets
  glance presets add <name> <regex> [desc]   Add user preset
  glance presets remove <name>               Remove user preset

Built-in presets cannot be removed or overridden.
`)
			fmt.Printf("User presets are stored in %s\n", configPath())
			return
		}
	}

	fmt.Print(`glance — LLM-optimized output summarizer

Pipe command output in, get a token-efficient summary showing head/tail
lines plus regex-matched lines, with an ID to drill into full output.

PIPE MODE:
  command | glance                  Head 10 + tail 10
  command | glance -n 5             Head 5 + tail 5
  command | glance -f 'ERROR|WARN'  + regex filter matches
  command | glance -p errors        + preset filter
  command | glance --no-store       Don't store, no ID

PIPE FLAGS:
  -n, --head N       Head/tail line count (default: 10)
  -f, --filter REGEX Additional middle-line filter (repeatable, OR)
  -p, --preset NAME  Named preset filter (repeatable, OR)
  --no-store         Don't store capture, no ID issued

SUBCOMMANDS:
  glance help [cmd]                    This help (or help for cmd)
  glance version                       Print version
  glance show <id>                     Full stored output
  glance show <id> -l 50-80            Line range
  glance show <id> -f 'regex'          Filter stored output
  glance show <id> -p errors           Filter with preset
  glance show <id> -a 247 5            Context around line
  glance list                          List stored captures
  glance clean                         Purge captures
  glance presets list                  Show all presets
  glance presets add <n> <re> [desc]   Add user preset
  glance presets remove <name>         Remove user preset

BUILT-IN PRESETS:
`)
	for _, p := range builtinPresets {
		fmt.Printf("  %-10s %s\n", p.name, p.regex)
	}
	fmt.Print(`
EXAMPLES:
  # Quick look at build output
  make 2>&1 | glance

  # Find errors in a long log
  kubectl logs pod/api | glance -p errors

  # Combine presets and custom filter
  docker compose up 2>&1 | glance -p errors -p status -f 'db:5432'

  # Drill into a specific capture (use full ID from footer)
  glance show 20260219-143022-a3f8b1c0 -a 247 5

PATHS:
`)
	fmt.Printf("  captures:  %s\n", cacheDir())
	fmt.Printf("  presets:   %s\n", configPath())
}
