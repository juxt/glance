package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

const defaultAroundContext = 5

type aroundSpec struct {
	center  int
	context int
}

type showConfig struct {
	id      string
	ranges  [][2]int
	around  []aroundSpec
	filters []string
}

func parseShowArgs(args []string) (showConfig, error) {
	if len(args) < 1 {
		return showConfig{}, fmt.Errorf("usage: glance show <id> [--lines N-M] [--filter regex] [--around N C]")
	}

	id := args[0]
	args = args[1:]

	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "..") {
		return showConfig{}, fmt.Errorf("invalid capture ID: %s", id)
	}

	cfg := showConfig{id: id}

	i := 0
	for i < len(args) {
		if parseFilter(args, &i, &cfg.filters) {
			continue
		}
		switch args[i] {
		case "-l", "--lines":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("invalid range, must be N-M")
			}
			start, end := parseRange(args[i+1])
			if start <= 0 || end <= 0 {
				return cfg, fmt.Errorf("invalid range, must be N-M")
			}
			cfg.ranges = append(cfg.ranges, [2]int{start, end})
			i += 2
		case "-a", "--around":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--around center must be a positive integer")
			}
			center := parsePositiveInt(args[i+1])
			if center <= 0 {
				return cfg, fmt.Errorf("--around center must be a positive integer")
			}
			ctx := defaultAroundContext
			if i+2 < len(args) && args[i+2] != "" && args[i+2][0] != '-' {
				ctx = parsePositiveInt(args[i+2])
				if ctx <= 0 {
					return cfg, fmt.Errorf("--around context must be a positive integer")
				}
				i += 3
			} else {
				i += 2
			}
			cfg.around = append(cfg.around, aroundSpec{center: center, context: ctx})
		default:
			return cfg, fmt.Errorf("unknown flag: %s", args[i])
		}
	}
	return cfg, nil
}

func doShow(args []string) {
	cfg, err := parseShowArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "glance show: %s\n", err)
		os.Exit(1)
	}
	runShow(cfg)
}

func runShow(cfg showConfig) {
	path := capturePath(cfg.id)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "glance: capture not found: %s\n", cfg.id)
		fmt.Fprintf(os.Stderr, "Use \"glance list\" to see stored captures.\n")
		os.Exit(1)
	}

	// No flags → dump full output
	if len(cfg.ranges) == 0 && len(cfg.around) == 0 && len(cfg.filters) == 0 {
		f, err := os.Open(path)
		if err != nil {
			fatal(err.Error())
		}
		defer f.Close()
		io.Copy(os.Stdout, f)
		return
	}

	// Precompute line numbers from ranges and around specs (no clamping to total)
	lineNums := make(map[int]bool)
	for _, r := range cfg.ranges {
		for j := r[0]; j <= r[1]; j++ {
			lineNums[j] = true
		}
	}
	for _, a := range cfg.around {
		from := a.center - a.context
		if from < 1 {
			from = 1
		}
		to := a.center + a.context
		for j := from; j <= to; j++ {
			lineNums[j] = true
		}
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

	// Single-pass scan
	f, err := os.Open(path)
	if err != nil {
		fatal(err.Error())
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, scanBufferSize), scanBufferSize)
	bw := bufio.NewWriter(os.Stdout)
	var printed []int
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		text := scanner.Text()
		show := lineNums[lineNo]
		if !show && re != nil {
			show = re.MatchString(text)
		}
		if show {
			fmt.Fprintf(bw, "%d: %s\n", lineNo, text)
			printed = append(printed, lineNo)
		}
	}
	if err := scanner.Err(); err != nil {
		fatal(err.Error())
	}

	total := lineNo
	sort.Ints(printed)
	sections := sectionRanges(printed)
	fmt.Fprintf(bw, "--- glance show %s | %s | showing %d | sections: %s ---\n", cfg.id, pluralLines(total), len(printed), sections)
	bw.Flush()
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
