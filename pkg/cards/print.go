package cards

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Fprint writes a stored card to w.
//
// A card is stored onchain as opaque bytes, which means the query APIs return it base64
// encoded inside a JSON envelope -- unreadable without a second decode step. This writes
// the decoded payload directly.
//
// When raw is false and the payload parses as JSON, it is re-indented for reading. When
// raw is true the exact stored bytes are written verbatim, which is what you want for
// hashing, diffing, or round-tripping a card back into a publish command.
//
// A payload that does not parse as JSON is always written verbatim: the chain never
// validated it, so refusing to display it would hide whatever is actually stored.
func Fprint(w io.Writer, card []byte, raw bool) error {
	if len(card) == 0 {
		return fmt.Errorf("no card is set")
	}

	if raw {
		_, err := w.Write(card)
		return err
	}

	var indented bytes.Buffer
	if err := json.Indent(&indented, card, "", "  "); err != nil {
		// Not JSON. Print what is stored rather than pretending it is absent.
		_, writeErr := w.Write(card)
		return writeErr
	}

	if _, err := indented.WriteTo(w); err != nil {
		return err
	}

	_, err := fmt.Fprintln(w)
	return err
}
