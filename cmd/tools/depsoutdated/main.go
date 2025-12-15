package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type module struct {
	Path     string
	Version  string
	Update   *moduleUpdate
	Indirect bool
	Main     bool
}

type moduleUpdate struct {
	Path    string
	Version string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "deps-outdated: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cmd := exec.Command("go", "list", "-m", "-u", "-json", "all")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("go list start: %w", err)
	}

	outdated, err := collectOutdated(stdout)
	if err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("go list wait: %w", err)
	}

	if len(outdated) == 0 {
		fmt.Println("すべて最新です")
		return nil
	}

	fmt.Printf("%-45s %-15s %-15s %s\n", "MODULE", "CURRENT", "LATEST", "NOTES")
	for _, m := range outdated {
		note := ""
		if m.Indirect {
			note = "(indirect)"
		}
		fmt.Printf("%-45s %-15s %-15s %s\n", m.Path, safeVersion(m.Version), safeVersion(m.Update.Version), note)
	}

	fmt.Println("\n更新例:")
	fmt.Println("  go get example.com/mod@v1.2.3")
	fmt.Println("  go mod tidy")

	return nil
}

func collectOutdated(r io.Reader) ([]module, error) {
	dec := json.NewDecoder(r)
	var outdated []module

	for {
		var m module
		if err := dec.Decode(&m); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode module: %w", err)
		}

		if m.Main || m.Update == nil {
			continue
		}

		outdated = append(outdated, m)
	}

	return outdated, nil
}

func safeVersion(v string) string {
	if v == "" {
		return "-"
	}
	return v
}
