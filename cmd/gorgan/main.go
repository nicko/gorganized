package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicko/gorganized/internal/tui"
)

const gorganDir = ".gorgan"

func main() {
	dir := filepath.Join(".", gorganDir)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Printf("%s/ not found. Create it here? [y/N] ", gorganDir)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" {
			fmt.Println("Aborted.")
			os.Exit(0)
		}
		if err := os.MkdirAll(filepath.Join(dir, "tasks"), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "error: could not create %s: %v\n", dir, err)
			os.Exit(1)
		}
		fmt.Printf("Created %s/\n", gorganDir)
	}

	if err := tui.Run(dir); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
