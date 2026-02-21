package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const defaultAroundContext = 5

func doShow(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: glance show <id> [--lines N-M] [--filter regex] [--around N C]\n")
		os.Exit(1)
	}

	id := args[0]
	args = args[1:]

	// Validate ID: no slashes, no .., not empty
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "..") {
		fmt.Fprintf(os.Stderr, "glance: invalid capture ID: %s\n", id)
		os.Exit(1)
	}

	path := capturePath(id)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "glance: capture not found: %s\n", id)
		fmt.Fprintf(os.Stderr, "Use \"glance list\" to see stored captures.\n")
		os.Exit(1)
	}

	// No flags → dump full output
	if len(args) == 0 {
		f, err := os.Open(path)
		if err != nil {
			fatal(err.Error())
		}
		defer f.Close()
		io.Copy(os.Stdout, f)
		return
	}

	// Count total lines
	total := countLines(path)

	lineNums := make(map[int]bool)
	var filters []string

	i := 0
	for i < len(args) {
		switch args[i] {
		case "-l", "--lines":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "glance show: invalid range, must be N-M\n")
				os.Exit(1)
			}
			start, end := parseRange(args[i+1])
			if start <= 0 || end <= 0 {
				fmt.Fprintf(os.Stderr, "glance show: invalid range, must be N-M\n")
				os.Exit(1)
			}
			if end > total {
				end = total
			}
			for j := start; j <= end; j++ {
				lineNums[j] = true
			}
			i += 2
		case "-f", "--filter":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "glance show: -f requires a value\n")
				os.Exit(1)
			}
			filters = append(filters, args[i+1])
			i += 2
		case "-p", "--preset":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "glance show: -p requires a value\n")
				os.Exit(1)
			}
			regex, err := resolvePreset(args[i+1])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			filters = append(filters, regex)
			i += 2
		case "-a", "--around":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "glance show: --around center must be a positive integer\n")
				os.Exit(1)
			}
			center := parsePositiveInt(args[i+1])
			if center <= 0 {
				fmt.Fprintf(os.Stderr, "glance show: --around center must be a positive integer\n")
				os.Exit(1)
			}
			ctx := defaultAroundContext
			if i+2 < len(args) && args[i+2] != "" && args[i+2][0] != '-' {
				ctx = parsePositiveInt(args[i+2])
				if ctx <= 0 {
					fmt.Fprintf(os.Stderr, "glance show: --around context must be a positive integer\n")
					os.Exit(1)
				}
				i += 3
			} else {
				i += 2
			}
			from := center - ctx
			if from < 1 {
				from = 1
			}
			to := center + ctx
			if to > total {
				to = total
			}
			for j := from; j <= to; j++ {
				lineNums[j] = true
			}
		default:
			fmt.Fprintf(os.Stderr, "glance show: unknown flag: %s\n", args[i])
			os.Exit(1)
		}
	}

	pattern := joinFilters(filters)
	footer := fmt.Sprintf("--- glance show %s | %s", id, pluralLines(total))

	if err := printMatchedLines(os.Stdout, path, lineNums, pattern, footer); err != nil {
		fmt.Fprintf(os.Stderr, "glance: %s\n", err)
		os.Exit(1)
	}
}

func parseRange(s string) (int, int) {
	idx := strings.Index(s, "-")
	if idx < 0 {
		return 0, 0
	}
	start := parsePositiveInt(s[:idx])
	end := parsePositiveInt(s[idx+1:])
	return start, end
}

func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	count := 0
	buf := make([]byte, 32*1024)
	for {
		n, err := f.Read(buf)
		for i := 0; i < n; i++ {
			if buf[i] == '\n' {
				count++
			}
		}
		if err != nil {
			break
		}
	}
	return count
}
