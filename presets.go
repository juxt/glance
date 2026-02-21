package main

import (
	"encoding/csv"
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
	{"errors", `(?i)error|err|fail|fatal|panic|exception|traceback`, "Error detection"},
	{"warnings", `(?i)warn|warning|deprecated`, "Warnings"},
	{"status", `(?i)exit code|status|returned?\s+[0-9]+|HTTP\s+[45][0-9][0-9]`, "Status/exit codes"},
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
	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	var result []preset
	for _, rec := range records {
		if len(rec) < 2 {
			continue
		}
		p := preset{name: rec[0], regex: rec[1]}
		if len(rec) >= 3 {
			p.desc = rec[2]
		}
		result = append(result, p)
	}
	return result, nil
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

func writePresets(path string, presets []preset) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	for _, p := range presets {
		if err := w.Write([]string{p.name, p.regex, p.desc}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
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
			fmt.Printf("  %-10s %-20s %s\n", p.name, p.desc, p.regex)
		}
		userPresets, _ := readUserPresets()
		if len(userPresets) > 0 {
			fmt.Println()
			fmt.Println("User presets:")
			for _, p := range userPresets {
				fmt.Printf("  %-10s %-20s %s\n", p.name, p.desc, p.regex)
			}
		}

	case "add":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: glance presets add <name> <regex> [description]\n")
			os.Exit(1)
		}
		name := args[0]
		regex := args[1]
		desc := ""
		if len(args) >= 3 {
			desc = strings.Join(args[2:], " ")
		}

		if !isValidPresetName(name) {
			fmt.Fprintf(os.Stderr, "glance: invalid preset name: %s (must start with alphanumeric, use only alphanumeric/hyphens/underscores)\n", name)
			os.Exit(1)
		}

		if _, ok := getBuiltinPreset(name); ok {
			fmt.Fprintf(os.Stderr, "glance: cannot override built-in preset: %s\n", name)
			os.Exit(1)
		}

		if err := ensureConfigDir(); err != nil {
			fatal(err.Error())
		}

		confPath := configPath()
		existing, _ := scanPresetFile(confPath)

		// Filter out same-name entry
		var kept []preset
		for _, p := range existing {
			if p.name != name {
				kept = append(kept, p)
			}
		}
		kept = append(kept, preset{name: name, regex: regex, desc: desc})

		if err := writePresets(confPath, kept); err != nil {
			fatal(err.Error())
		}

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
		existing, err := scanPresetFile(confPath)
		if err != nil || existing == nil {
			fmt.Fprintf(os.Stderr, "glance: preset not found: %s\n", name)
			os.Exit(1)
		}

		var kept []preset
		found := false
		for _, p := range existing {
			if p.name == name {
				found = true
				continue
			}
			kept = append(kept, p)
		}

		if !found {
			fmt.Fprintf(os.Stderr, "glance: preset not found: %s\n", name)
			os.Exit(1)
		}

		if err := writePresets(confPath, kept); err != nil {
			fatal(err.Error())
		}

		fmt.Printf("Removed preset: %s\n", name)

	default:
		fmt.Fprintf(os.Stderr, "glance presets: unknown subcommand: %s\n", sub)
		os.Exit(1)
	}
}
