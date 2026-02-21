package main

import (
	"fmt"
	"os"
)

func doClean(args []string) {
	if len(args) > 0 && args[0] == "--all" {
		os.RemoveAll(cacheDir())
		confPath := configPath()
		if _, err := os.Stat(confPath); err == nil {
			os.Remove(confPath)
		}
		fmt.Println("Purged all captures and user presets.")
	} else {
		os.RemoveAll(cacheDir())
		fmt.Println("Purged all captures.")
	}
}
