// Command configmd renders generated doc fragments from internal/core
// metadata. Invoked by the cog markers embedded in
// docs/user/messengers.md and docs/dev/configuration.md (see
// `make docs-gen`); never run in production paths.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"hop.top/aps/internal/core"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func run(args []string, w io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: configmd secret-backends|config-fields")
	}
	switch args[0] {
	case "secret-backends":
		return secretBackends(w)
	case "config-fields":
		return configFields(w)
	default:
		return fmt.Errorf("unknown fragment %q", args[0])
	}
}

// secretBackends renders the "Secret Store Backends" table in
// docs/user/messengers.md. The † marks backends that are opt-in at
// build time; the footnote below the table is hand-written.
func secretBackends(w io.Writer) error {
	rows := make([]string, 0, 2+len(core.SecretsBackendsMeta))
	rows = append(rows,
		"| `secrets.backend` | Where secrets live | Required config |",
		"| --- | --- | --- |")
	for _, m := range core.SecretsBackendsMeta {
		name := "`" + m.Name + "`"
		if m.Default {
			name += " (default)"
		}
		if m.BuildTag != "" {
			name += " †"
		}
		rows = append(rows, fmt.Sprintf("| %s | %s | %s |", name, m.Where, m.Required))
	}
	return writeTable(w, rows)
}

// configFields renders the "Fields" table in docs/dev/configuration.md.
func configFields(w io.Writer) error {
	docs, err := core.ConfigFieldDocs()
	if err != nil {
		return fmt.Errorf("config fields: %w", err)
	}
	rows := make([]string, 0, 2+len(docs))
	rows = append(rows,
		"| Field | Default | Description |",
		"|-------|---------|-------------|")
	for _, d := range docs {
		rows = append(rows, fmt.Sprintf("| `%s` | %s | %s |", d.Path, d.Default, d.Description))
	}
	return writeTable(w, rows)
}

// writeTable emits rows as newline-terminated lines.
func writeTable(w io.Writer, rows []string) error {
	if _, err := io.WriteString(w, strings.Join(rows, "\n")+"\n"); err != nil {
		return fmt.Errorf("writing fragment: %w", err)
	}
	return nil
}
