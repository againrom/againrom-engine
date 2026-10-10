package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"againrom/pkg/graphcopy"
)

// One load and save path reads a document's current-action supplement many
// times over. The JSON decode is the same each time, so the last few decoded
// supplements are kept by their exact bytes and each reader receives its own
// deep copy; validation against the document still runs on every read.
const currentActionDecodeEntries = 4

type currentActionDecoded struct {
	raw   string
	value *currentActionData
}

var currentActionDecodes struct {
	mu      sync.Mutex
	entries []currentActionDecoded
}

// decodeCurrentActions decodes b strictly into a fresh value that shares no
// memory with any other caller's value.
func decodeCurrentActions(b []byte) (*currentActionData, error) {
	currentActionDecodes.mu.Lock()
	for i, e := range currentActionDecodes.entries {
		if e.raw == string(b) {
			if out, ok := graphcopy.Clone(e.value); ok {
				copy(currentActionDecodes.entries[1:i+1], currentActionDecodes.entries[:i])
				currentActionDecodes.entries[0] = e
				currentActionDecodes.mu.Unlock()
				return out, nil
			}
			break
		}
	}
	currentActionDecodes.mu.Unlock()
	var a currentActionData
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&a); err != nil {
		return nil, fmt.Errorf("current actions: %w", err)
	}
	var tail any
	if err := d.Decode(&tail); err != io.EOF {
		return nil, fmt.Errorf("current actions contain trailing data")
	}
	if kept, ok := graphcopy.Clone(&a); ok {
		currentActionDecodes.mu.Lock()
		entries := append([]currentActionDecoded{{raw: string(b), value: kept}}, currentActionDecodes.entries...)
		currentActionDecodes.entries = entries[:min(len(entries), currentActionDecodeEntries)]
		currentActionDecodes.mu.Unlock()
	}
	return &a, nil
}
