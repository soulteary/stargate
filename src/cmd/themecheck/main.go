// themecheck validates native authentication theme directories without a server.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/soulteary/stargate/src/internal/theme"
)

func run(args []string, out io.Writer) int {
	jsonOutput := len(args) > 0 && args[0] == "-json"
	if jsonOutput {
		args = args[1:]
	}
	if len(args) == 0 {
		_, _ = fmt.Fprintln(out, "usage: themecheck [-json] DIRECTORY...")
		return 2
	}
	status := 0
	entries := []map[string]any{}
	for _, directory := range args {
		selected, err := theme.Load(directory)
		if err != nil {
			_, _ = fmt.Fprintf(out, "FAIL %s: %v\n", directory, err)
			status = 1
			continue
		}
		if jsonOutput {
			entries = append(entries, map[string]any{"id": selected.ID(), "name": selected.Name(), "version": selected.Version(), "author": selected.Author(), "license": selected.License(), "contract": theme.Contract, "ui_contract": theme.Contract, "digest": strings.TrimPrefix(selected.BaseURL(), "/_theme/")})
		} else {
			_, _ = fmt.Fprintf(out, "ok %s (%s %s, %s)\n", selected.ID(), selected.Name(), selected.Version(), selected.BaseURL())
		}
	}
	if jsonOutput && status == 0 {
		if err := json.NewEncoder(out).Encode(entries); err != nil {
			return 1
		}
	}
	return status
}

func main() { os.Exit(run(os.Args[1:], os.Stdout)) }
