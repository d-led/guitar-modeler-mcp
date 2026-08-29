package rig

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
)

// TestMarshalIsSingleLineJSON guards the on-disk format: the device writes
// rig files as one line of minified JSON with no newline characters.
func TestMarshalIsSingleLineJSON(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name: "Single Line",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	out, err := file.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if bytes.ContainsAny(out, "\n\r") {
		t.Fatalf("rig JSON must not contain newlines:\n%s", out)
	}
	if !json.Valid(out) {
		t.Fatal("rig JSON is not valid")
	}
}

// TestWriteProducesSingleLineFile verifies the file on disk matches the device
// format: a single line with no trailing newline.
func TestWriteProducesSingleLineFile(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name: "Single Line",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	path, err := file.Write(t.TempDir())
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if bytes.ContainsAny(raw, "\n\r") {
		t.Fatalf("written rig must be one line, got %d bytes with newlines", len(raw))
	}
}
