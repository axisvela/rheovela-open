// Package conformance exposes public conformance corpora.
package conformance

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
)

// RHEOIRv1 is the embedded RHEO IR v1.0 conformance corpus.
//
//go:embed rheo-ir/v1.0
var RHEOIRv1 embed.FS

// RHEOIRv1Root is the directory of the RHEO IR v1.0 corpus inside RHEOIRv1.
const RHEOIRv1Root = "rheo-ir/v1.0"

// Case names one fixture pair.
type Case struct {
	Name     string `json:"name"`
	Source   string `json:"source"`
	Expected string `json:"expected"`
}

// Manifest describes the RHEO IR v1.0 conformance corpus.
type Manifest struct {
	Schema  string `json:"schema"`
	Version string `json:"version"`
	Cases   struct {
		Valid   []Case `json:"valid"`
		Invalid []Case `json:"invalid"`
	} `json:"cases"`
}

// LoadManifest reads and validates the manifest from fsys.
func LoadManifest(fsys fs.FS) (*Manifest, error) {
	b, err := fs.ReadFile(fsys, RHEOIRv1Root+"/manifest.json")
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	if m.Schema != "rheo-ir-conformance" {
		return nil, fmt.Errorf("unexpected manifest schema %q", m.Schema)
	}
	if m.Version != "1.0" {
		return nil, fmt.Errorf("unexpected manifest version %q", m.Version)
	}
	return &m, nil
}
