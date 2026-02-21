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

func formatAge(secs int64) string {
	if secs <= 0 {
		return "unknown"
	}
	if secs < 60 {
		return fmt.Sprintf("%ds ago", secs)
	}
	if secs < 3600 {
		return fmt.Sprintf("%dm ago", secs/60)
	}
	if secs < 86400 {
		return fmt.Sprintf("%dh ago", secs/3600)
	}
	return fmt.Sprintf("%dd ago", secs/86400)
}

func pluralLines(n int) string {
	if n == 1 {
		return "1 line"
	}
	return fmt.Sprintf("%d lines", n)
}

// sectionRanges takes sorted line numbers and returns a string like "1-5, 10, 20-25"
func sectionRanges(nums []int) string {
	if len(nums) == 0 {
		return ""
	}
	var parts []string
	start := nums[0]
	prev := nums[0]
	for i := 1; i < len(nums); i++ {
		if nums[i] == prev+1 {
			prev = nums[i]
			continue
		}
		if start == prev {
			parts = append(parts, fmt.Sprintf("%d", start))
		} else {
			parts = append(parts, fmt.Sprintf("%d-%d", start, prev))
		}
		start = nums[i]
		prev = nums[i]
	}
	if start == prev {
		parts = append(parts, fmt.Sprintf("%d", start))
	} else {
		parts = append(parts, fmt.Sprintf("%d-%d", start, prev))
	}
	return strings.Join(parts, ", ")
}

// printMatchedLines prints lines from file whose numbers are in lineNums set
// or that match pattern (case-insensitive). If footer is non-empty, appends summary.
const scanBufferSize = 1024 * 1024

func printMatchedLines(w io.Writer, filePath string, lineNums map[int]bool, pattern string, footer string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Count total lines
	total := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, scanBufferSize), scanBufferSize)
	for scanner.Scan() {
		total++
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	width := len(fmt.Sprintf("%d", total))
	if width == 0 {
		width = 1
	}

	// Compile regex if given
	var re *regexp.Regexp
	if pattern != "" {
		re, err = regexp.Compile("(?i)" + pattern)
		if err != nil {
			return fmt.Errorf("invalid regex %s: %w", pattern, err)
		}
	}

	// Second pass: print matching lines
	f.Seek(0, 0)
	scanner = bufio.NewScanner(f)
	scanner.Buffer(make([]byte, scanBufferSize), scanBufferSize)
	lineNo := 0
	var matched []int
	bw := bufio.NewWriter(w)
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		show := lineNums[lineNo]
		if !show && re != nil {
			show = re.MatchString(line)
		}
		if show {
			fmt.Fprintf(bw, "%*d: %s\n", width, lineNo, line)
			matched = append(matched, lineNo)
		}
	}

	if footer != "" {
		sort.Ints(matched)
		sections := sectionRanges(matched)
		fmt.Fprintf(bw, "%s | showing %d | sections: %s ---\n", footer, len(matched), sections)
	}

	bw.Flush()
	return scanner.Err()
}
