// Package output formats CLI results as table, JSON, or YAML.
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

// Format is the global output mode.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

var current = FormatTable

// SetFormat sets the global output format.
func SetFormat(f string) error {
	switch Format(f) {
	case FormatTable, FormatJSON, FormatYAML:
		current = Format(f)
		return nil
	default:
		return fmt.Errorf("output must be table, json, or yaml")
	}
}

// GetFormat returns the active format.
func GetFormat() Format { return current }

func keyify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

// Table prints columns/rows in the active format.
func Table(columns []string, rows [][]string, title string) error {
	switch current {
	case FormatJSON:
		items := make([]map[string]string, 0, len(rows))
		keys := make([]string, len(columns))
		for i, c := range columns {
			keys[i] = keyify(c)
		}
		for _, row := range rows {
			m := map[string]string{}
			for i, k := range keys {
				if i < len(row) {
					m[k] = row[i]
				}
			}
			items = append(items, m)
		}
		return encodeJSON(items)
	case FormatYAML:
		keys := make([]string, len(columns))
		for i, c := range columns {
			keys[i] = keyify(c)
		}
		items := make([]map[string]string, 0, len(rows))
		for _, row := range rows {
			m := map[string]string{}
			for i, k := range keys {
				if i < len(row) {
					m[k] = row[i]
				}
			}
			items = append(items, m)
		}
		return encodeYAML(items)
	default:
		if title != "" {
			fmt.Fprintln(os.Stdout, title)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, strings.Join(columns, "\t"))
		for _, row := range rows {
			fmt.Fprintln(w, strings.Join(row, "\t"))
		}
		return w.Flush()
	}
}

// KV prints key/value pairs.
func KV(pairs [][2]string, title string) error {
	switch current {
	case FormatJSON:
		m := map[string]string{}
		for _, p := range pairs {
			m[keyify(p[0])] = p[1]
		}
		return encodeJSON(m)
	case FormatYAML:
		m := map[string]string{}
		for _, p := range pairs {
			m[keyify(p[0])] = p[1]
		}
		return encodeYAML(m)
	default:
		if title != "" {
			fmt.Fprintln(os.Stdout, title)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		for _, p := range pairs {
			fmt.Fprintf(w, "%s\t%s\n", p[0], p[1])
		}
		return w.Flush()
	}
}

// Raw writes raw JSON for -o json/yaml, otherwise returns false so caller can print a table.
func Raw(raw json.RawMessage) (handled bool, err error) {
	switch current {
	case FormatJSON:
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return true, err
		}
		return true, encodeJSON(v)
	case FormatYAML:
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return true, err
		}
		return true, encodeYAML(v)
	default:
		return false, nil
	}
}

// Success prints a success message (JSON object when -o json).
func Success(msg string, extra map[string]any) error {
	if current == FormatJSON {
		m := map[string]any{"status": "ok", "message": msg}
		for k, v := range extra {
			m[k] = v
		}
		return encodeJSON(m)
	}
	if current == FormatYAML {
		m := map[string]any{"status": "ok", "message": msg}
		for k, v := range extra {
			m[k] = v
		}
		return encodeYAML(m)
	}
	fmt.Fprintf(os.Stdout, "✓ %s\n", msg)
	return nil
}

// Errorf prints an error to stderr.
func Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

func encodeJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func encodeYAML(v any) error {
	b, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(b)
	return err
}
