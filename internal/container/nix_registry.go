package container

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type nixRegistryDocument struct {
	Version int                `json:"version"`
	Flakes  []nixRegistryAlias `json:"flakes"`
}

type nixRegistryAlias struct {
	From json.RawMessage `json:"from"`
	To   json.RawMessage `json:"to"`
}

// MergeNixRegistryAliases preserves existing aliases and appends aliases from
// incomingJSON only when their "from" key is missing in existingJSON.
func MergeNixRegistryAliases(existingJSON, incomingJSON []byte, existingPath, incomingPath string) ([]byte, bool, error) {
	existingDoc, err := parseNixRegistryJSON(existingJSON, existingPath)
	if err != nil {
		return nil, false, err
	}
	incomingDoc, err := parseNixRegistryJSON(incomingJSON, incomingPath)
	if err != nil {
		return nil, false, err
	}

	keys := make(map[string]struct{}, len(existingDoc.Flakes))
	for _, alias := range existingDoc.Flakes {
		key := string(bytes.TrimSpace(alias.From))
		if key == "" {
			continue
		}
		keys[key] = struct{}{}
	}

	changed := false
	for _, alias := range incomingDoc.Flakes {
		key := string(bytes.TrimSpace(alias.From))
		if key == "" {
			continue
		}
		if _, exists := keys[key]; exists {
			continue
		}
		existingDoc.Flakes = append(existingDoc.Flakes, alias)
		keys[key] = struct{}{}
		changed = true
	}

	if !changed {
		return existingJSON, false, nil
	}

	out, err := json.Marshal(existingDoc)
	if err != nil {
		return nil, false, fmt.Errorf("encode merged nix registry aliases: %w", err)
	}

	return out, true, nil
}

func parseNixRegistryJSON(data []byte, path string) (nixRegistryDocument, error) {
	var doc nixRegistryDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nixRegistryDocument{}, fmt.Errorf("parse nix registry file %q: %w (fix JSON syntax or remove the file to let havn recreate it)", path, err)
	}
	if doc.Flakes == nil {
		doc.Flakes = []nixRegistryAlias{}
	}
	return doc, nil
}
