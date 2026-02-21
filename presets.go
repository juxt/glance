package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type preset struct {
	name  string
	regex string
	desc  string
}

var builtinPresets = []preset{
	{"errors", `error|err|fail|fatal|panic|exception|traceback`, "Error detection"},
	{"warnings", `warn|warning|deprecated`, "Warnings"},
	{"status", `exit code|status|returned?\s+[0-9]+|HTTP\s+[45][0-9][0-9]`, "Status/exit codes"},
}

// parsePresetLine parses a sed-style line: first char is delimiter, then name/regex/desc
func parsePresetLine(line string) (preset, error) {
	if len(line) < 2 {
		return preset{}, fmt.Errorf("preset line too short")
	}
	delim := string(line[0])
	rest := line[1:]
	parts := strings.SplitN(rest, delim, 3)
	if len(parts) < 2 {
		return preset{}, fmt.Errorf("invalid preset line")
	}
	p := preset{name: parts[0], regex: parts[1]}
	if len(parts) >= 3 {
		p.desc = parts[2]
	}
	return p, nil
}

func getBuiltinPreset(name string) (string, bool) {
	for _, p := range builtinPresets {
		if p.name == name {
			return p.regex, true
		}
	}
	return "", false
}

func scanPresetFile(path string) ([]preset, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var result []preset
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p, err := parsePresetLine(line)
		if err != nil {
			continue
		}
		result = append(result, p)
	}
	return result, scanner.Err()
}

func getUserPreset(name string) (string, bool) {
	presets, err := scanPresetFile(configPath())
	if err != nil {
		return "", false
	}
	for _, p := range presets {
		if p.name == name {
			return p.regex, true
		}
	}
	return "", false
}

func resolvePreset(name string) (string, error) {
	if r, ok := getBuiltinPreset(name); ok {
		return r, nil
	}
	if r, ok := getUserPreset(name); ok {
		return r, nil
	}
	return "", fmt.Errorf("glance: unknown preset: %s", name)
}

func readUserPresets() ([]preset, error) {
	return scanPresetFile(configPath())
}

func isValidPresetName(name string) bool {
	if name == "" {
		return false
	}
	if name[0] == '-' {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func doPresets(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: glance presets <list|add|remove>\n")
		os.Exit(1)
	}

	sub := args[0]
	args = args[1:]

	switch sub {
	case "list":
		fmt.Println("Built-in presets:")
		for _, p := range builtinPresets {
			fmt.Printf("  %-10s  %-50s  %s\n", p.name, p.regex, p.desc)
		}
		userPresets, _ := readUserPresets()
		if len(userPresets) > 0 {
			fmt.Println()
			fmt.Println("User presets:")
			for _, p := range userPresets {
				fmt.Printf("  %-10s  %-50s  %s\n", p.name, p.regex, p.desc)
			}
		} else {
			// Check if config file exists — might have presets
			path := configPath()
			if f, err := os.Open(path); err == nil {
				f.Close()
				// File exists but no parseable presets — still show section if file has content
			}
		}

	case "add":
		delim := ""
		// Parse optional -d flag
		if len(args) >= 2 && args[0] == "-d" {
			delim = args[1]
			args = args[2:]
		}
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: glance presets add [-d delim] <name> <regex> [description]\n")
			os.Exit(1)
		}
		name := args[0]
		regex := args[1]
		desc := ""
		if len(args) >= 3 {
			desc = args[2]
		}

		if !isValidPresetName(name) {
			fmt.Fprintf(os.Stderr, "glance: invalid preset name: %s (must start with alphanumeric, use only alphanumeric/hyphens/underscores)\n", name)
			os.Exit(1)
		}

		if _, ok := getBuiltinPreset(name); ok {
			fmt.Fprintf(os.Stderr, "glance: cannot override built-in preset: %s\n", name)
			os.Exit(1)
		}

		// Pick delimiter
		if delim != "" {
			if strings.Contains(regex, delim) {
				fmt.Fprintf(os.Stderr, "glance: delimiter %s appears in regex, choose another\n", delim)
				os.Exit(1)
			}
		} else {
			for _, c := range []string{"/", ",", "@", "#", "%", "~", "!"} {
				if !strings.Contains(regex, c) {
					delim = c
					break
				}
			}
			if delim == "" {
				fmt.Fprintf(os.Stderr, "glance: regex contains all candidate delimiters, use -d to specify one\n")
				os.Exit(1)
			}
		}

		if err := ensureConfigDir(); err != nil {
			fatal(err.Error())
		}

		// Read existing config, filter out same-name entry
		confPath := configPath()
		var lines []string
		if f, err := os.Open(confPath); err == nil {
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" || strings.HasPrefix(line, "#") {
					lines = append(lines, line)
					continue
				}
				p, err := parsePresetLine(line)
				if err != nil || p.name == name {
					continue
				}
				lines = append(lines, line)
			}
			f.Close()
		}

		// Append new entry
		newLine := fmt.Sprintf("%s%s%s%s%s%s", delim, name, delim, regex, delim, desc)
		lines = append(lines, newLine)

		// Write back
		f, err := os.Create(confPath)
		if err != nil {
			fatal(err.Error())
		}
		for _, l := range lines {
			fmt.Fprintln(f, l)
		}
		f.Close()

		fmt.Printf("Added preset: %s\n", name)

	case "remove":
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "Usage: glance presets remove <name>\n")
			os.Exit(1)
		}
		name := args[0]

		if _, ok := getBuiltinPreset(name); ok {
			fmt.Fprintf(os.Stderr, "glance: cannot remove built-in preset: %s\n", name)
			os.Exit(1)
		}

		confPath := configPath()
		f, err := os.Open(confPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "glance: preset not found: %s\n", name)
			os.Exit(1)
		}

		var lines []string
		found := false
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" || strings.HasPrefix(line, "#") {
				lines = append(lines, line)
				continue
			}
			p, err := parsePresetLine(line)
			if err != nil {
				lines = append(lines, line)
				continue
			}
			if p.name == name {
				found = true
				continue
			}
			lines = append(lines, line)
		}
		f.Close()

		if !found {
			fmt.Fprintf(os.Stderr, "glance: preset not found: %s\n", name)
			os.Exit(1)
		}

		out, err := os.Create(confPath)
		if err != nil {
			fatal(err.Error())
		}
		for _, l := range lines {
			fmt.Fprintln(out, l)
		}
		out.Close()

		fmt.Printf("Removed preset: %s\n", name)

	default:
		fmt.Fprintf(os.Stderr, "glance presets: unknown subcommand: %s\n", sub)
		os.Exit(1)
	}
}
