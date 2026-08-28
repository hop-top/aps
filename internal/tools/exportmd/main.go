// Command exportmd renders the `aps profile export` format
// enumeration from the registry declared in internal/cli
// (profileExportFormats via ExportFormatDocs). Invoked by the cog
// markers embedded in docs/cli/reference.md (see `make docs-gen`);
// never run in production paths.
package main

import (
	"fmt"
	"os"

	"hop.top/aps/internal/cli"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "formats-table" {
		fmt.Fprintln(os.Stderr, "usage: exportmd formats-table")
		os.Exit(2)
	}
	fmt.Println("| Format | Rendering | Emits |")
	fmt.Println("|--------|-----------|-------|")
	for _, f := range cli.ExportFormatDocs() {
		flag := "*(flag omitted)*"
		if f.Flag != "" {
			flag = fmt.Sprintf("`--manifest-format %s`", f.Flag)
		}
		fmt.Printf("| %s | %s | %s |\n", flag, f.Summary, f.Emits)
	}
}
