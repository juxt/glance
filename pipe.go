package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"sort"
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

type pipeConfig struct {
	n       int
	filters []string
	noStore bool
}

func parsePipeArgs(args []string) (pipeConfig, error) {
	cfg := pipeConfig{n: defaultHeadTail}

	i := 0
	for i < len(args) {
		if parseFilter(args, &i, &cfg.filters) {
			continue
		}
		switch args[i] {
		case "-n", "--lines", "--head":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("-n must be a positive integer")
			}
			v := parsePositiveInt(args[i+1])
			if v <= 0 {
				return cfg, fmt.Errorf("-n must be a positive integer")
			}
			cfg.n = v
			i += 2
		case "--no-store":
			cfg.noStore = true
			i++
		default:
			return cfg, fmt.Errorf("unknown flag: %s", args[i])
		}
	}
	return cfg, nil
}

func doPipe(args []string) {
	cfg, err := parsePipeArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "glance: %s\n", err)
		if strings.Contains(err.Error(), "unknown flag") {
			fmt.Fprintf(os.Stderr, "Try: glance help\n")
		}
		os.Exit(1)
	}
	runPipe(cfg)
}

func runPipe(cfg pipeConfig) {
	// Open capture file if storing
	var captureID string
	var captureW *bufio.Writer
	var captureF *os.File
	if !cfg.noStore {
		if err := ensureCacheDir(); err != nil {
			fatal(err.Error())
		}
		captureID = genID()
		var err error
		captureF, err = os.Create(capturePath(captureID))
		if err != nil {
			fatal(err.Error())
		}
		defer captureF.Close()
		captureW = bufio.NewWriter(captureF)
	}

	// Compile regex
	pattern := joinFilters(cfg.filters)
	var re *regexp.Regexp
	if pattern != "" {
		var err error
		re, err = regexp.Compile("(?i)" + pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "glance: invalid regex %s: %s\n", pattern, err)
			os.Exit(1)
		}
	}

	n := cfg.n
	ring := newRingBuffer(n)
	bw := bufio.NewWriter(os.Stdout)
	var printed []int
	lineNo := 0

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, scanBufferSize), scanBufferSize)
	for scanner.Scan() {
		lineNo++
		text := scanner.Text()

		// Write to capture file
		if captureW != nil {
			captureW.WriteString(text)
			captureW.WriteByte('\n')
		}

		if lineNo <= n {
			// Head: print eagerly
			fmt.Fprintf(bw, "%d: %s\n", lineNo, text)
			printed = append(printed, lineNo)
			continue
		}

		// Past head: use ring buffer
		matched := re != nil && re.MatchString(text)
		evicted, ok := ring.push(lineNo, text, matched)
		if ok && evicted.matched {
			// Evicted middle match: print it
			fmt.Fprintf(bw, "%d: %s\n", evicted.num, evicted.text)
			printed = append(printed, evicted.num)
		}
	}
	if err := scanner.Err(); err != nil {
		fatal(err.Error())
	}

	// Flush capture
	if captureW != nil {
		captureW.Flush()
	}

	total := lineNo

	// Print tail from ring buffer
	for _, e := range ring.entries() {
		fmt.Fprintf(bw, "%d: %s\n", e.num, e.text)
		printed = append(printed, e.num)
	}

	// Footer
	sort.Ints(printed)
	sections := sectionRanges(printed)
	if captureID != "" {
		if total == 0 {
			fmt.Fprintf(bw, "--- glance id=%s | 0 lines | showing 0 ---\n", captureID)
		} else {
			fmt.Fprintf(bw, "--- glance id=%s | %s | showing %d | sections: %s ---\n", captureID, pluralLines(total), len(printed), sections)
		}
	} else {
		if total == 0 {
			fmt.Fprintf(bw, "--- glance | 0 lines | showing 0 ---\n")
		} else {
			fmt.Fprintf(bw, "--- glance | %s | showing %d | sections: %s ---\n", pluralLines(total), len(printed), sections)
		}
	}
	bw.Flush()
}

// ringEntry holds a buffered line for the tail window.
type ringEntry struct {
	num     int
	text    string
	matched bool
}

// ringBuffer is a fixed-size circular buffer for tail lines.
type ringBuffer struct {
	buf  []ringEntry
	pos  int
	full bool
}

func newRingBuffer(n int) *ringBuffer {
	return &ringBuffer{buf: make([]ringEntry, n)}
}

// push adds an entry, returning the evicted entry (if any).
func (r *ringBuffer) push(num int, text string, matched bool) (ringEntry, bool) {
	if len(r.buf) == 0 {
		return ringEntry{num: num, text: text, matched: matched}, true
	}
	var evicted ringEntry
	var didEvict bool
	if r.full {
		evicted = r.buf[r.pos]
		didEvict = true
	}
	r.buf[r.pos] = ringEntry{num: num, text: text, matched: matched}
	r.pos++
	if r.pos == len(r.buf) {
		r.pos = 0
		r.full = true
	}
	return evicted, didEvict
}

// entries returns all buffered entries in insertion order.
func (r *ringBuffer) entries() []ringEntry {
	if !r.full {
		return r.buf[:r.pos]
	}
	out := make([]ringEntry, len(r.buf))
	copy(out, r.buf[r.pos:])
	copy(out[len(r.buf)-r.pos:], r.buf[:r.pos])
	return out
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
