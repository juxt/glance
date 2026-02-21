package main

import (
	"fmt"
	"os"
)

// consumeFlag checks that args[i+1] exists and returns it, advancing i by 2.
// If missing, prints an error and exits.
func consumeFlag(args []string, i *int, name string) string {
	if *i+1 >= len(args) {
		fmt.Fprintf(os.Stderr, "glance: %s requires a value\n", name)
		os.Exit(1)
	}
	v := args[*i+1]
	*i += 2
	return v
}

// parseFilters handles -f/--filter and -p/--preset flags, appending to filters.
// Returns true if the flag was consumed, false otherwise.
func parseFilter(args []string, i *int, filters *[]string) bool {
	switch args[*i] {
	case "-f", "--filter":
		v := consumeFlag(args, i, "-f")
		*filters = append(*filters, v)
		return true
	case "-p", "--preset":
		v := consumeFlag(args, i, "-p")
		regex, err := resolvePreset(v)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		*filters = append(*filters, regex)
		return true
	}
	return false
}
