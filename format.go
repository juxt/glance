package main

import (
	"fmt"
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

const scanBufferSize = 1024 * 1024
