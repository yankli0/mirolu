package ui

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
)

// WriteJSON encodes v as deterministic indented JSON followed by a newline.
func WriteJSON(w io.Writer, v any) error {
	if err := json.MarshalWrite(w, v, json.Deterministic(true), jsontext.WithIndent("  ")); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}
