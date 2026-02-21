package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func doList() {
	if err := ensureCacheDir(); err != nil {
		fatal(err.Error())
	}

	entries, err := os.ReadDir(cacheDir())
	if err != nil {
		fatal(err.Error())
	}

	// Sort by name (which is timestamp-based)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	found := false
	now := time.Now()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		found = true
		id := strings.TrimSuffix(e.Name(), ".txt")
		path := filepath.Join(cacheDir(), e.Name())
		lines := countLines(path)
		info, err := e.Info()
		ageStr := "unknown"
		if err == nil {
			secs := int64(now.Sub(info.ModTime()).Seconds())
			ageStr = formatAge(secs)
		}
		fmt.Printf("%s\t%d lines\t%s\n", id, lines, ageStr)
	}
	if !found {
		fmt.Println("No stored captures.")
	}
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
