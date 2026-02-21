package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

func genID() string {
	ts := time.Now().Format("20060102-150405")
	b := make([]byte, 4)
	rand.Read(b)
	return ts + "-" + hex.EncodeToString(b)
}

const defaultHeadTail = 10

func doPipe(args []string) {
	n := defaultHeadTail
	var filters []string
	noStore := false

	i := 0
	for i < len(args) {
		switch args[i] {
		case "-n", "--lines", "--head":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "glance: -n must be a positive integer\n")
				os.Exit(1)
			}
			v := parsePositiveInt(args[i+1])
			if v <= 0 {
				fmt.Fprintf(os.Stderr, "glance: -n must be a positive integer\n")
				os.Exit(1)
			}
			n = v
			i += 2
		case "-f", "--filter":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "glance: -f requires a value\n")
				os.Exit(1)
			}
			filters = append(filters, args[i+1])
			i += 2
		case "-p", "--preset":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "glance: -p requires a value\n")
				os.Exit(1)
			}
			regex, err := resolvePreset(args[i+1])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			filters = append(filters, regex)
			i += 2
		case "--no-store":
			noStore = true
			i++
		default:
			fmt.Fprintf(os.Stderr, "glance: unknown flag: %s\n", args[i])
			fmt.Fprintf(os.Stderr, "Try: glance help\n")
			os.Exit(1)
		}
	}

	// Buffer stdin to temp file
	tmpfile, err := os.CreateTemp("", "glance-*")
	if err != nil {
		fatal(err.Error())
	}
	defer os.Remove(tmpfile.Name())

	// Copy stdin and count lines
	total := 0
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(tmpfile)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			writer.Write(line)
			total++
			// If line doesn't end with \n, it's still a line
			if len(line) > 0 && line[len(line)-1] != '\n' {
				writer.WriteByte('\n')
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fatal(err.Error())
		}
	}
	writer.Flush()
	tmpfile.Close()

	// Store capture
	captureID := ""
	if !noStore {
		if err := ensureCacheDir(); err != nil {
			fatal(err.Error())
		}
		captureID = genID()
		copyFile(tmpfile.Name(), capturePath(captureID))
	}

	// Empty input
	if total == 0 {
		if captureID != "" {
			fmt.Printf("--- glance id=%s | 0 lines | showing 0 ---\n", captureID)
		} else {
			fmt.Println("--- glance | 0 lines | showing 0 ---")
		}
		return
	}

	// Calculate head/tail
	headEnd := n
	tailStart := total - n + 1
	if total <= n*2 {
		headEnd = total
		tailStart = total + 1
	}

	lineNums := make(map[int]bool)
	for j := 1; j <= headEnd; j++ {
		lineNums[j] = true
	}
	if tailStart <= total {
		for j := tailStart; j <= total; j++ {
			lineNums[j] = true
		}
	}

	// Build filter pattern
	pattern := joinFilters(filters)

	// Build footer
	var footer string
	if captureID != "" {
		footer = fmt.Sprintf("--- glance id=%s | %s", captureID, pluralLines(total))
	} else {
		footer = fmt.Sprintf("--- glance | %s", pluralLines(total))
	}

	if err := printMatchedLines(os.Stdout, tmpfile.Name(), lineNums, pattern, footer); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func joinFilters(filters []string) string {
	if len(filters) == 0 {
		return ""
	}
	return strings.Join(filters, "|")
}

func parsePositiveInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func copyFile(src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		fatal(err.Error())
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		fatal(err.Error())
	}
	defer out.Close()
	io.Copy(out, in)
}
