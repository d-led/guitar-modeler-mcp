package qc

import (
	"encoding/json"
	"fmt"
)

// presetJSONFormat labels the .json sidecar so it can never be mistaken for a
// device or upload format.
const presetJSONFormat = "guitar-modeler-mcp.qc-preset-v1"

// presetJSONNote is printed in every JSON view to keep its role honest.
const presetJSONNote = "Human-readable view of the decoded preset. This is " +
	"guitar-modeler-mcp's own representation — it is not a file the Quad " +
	"Cortex imports, and it is not a qcctl format. The .pb is the lossless " +
	"archive; reproduce the tone from the HTML card."

// PresetView is the human-readable JSON view of a decoded preset: the
// metadata, the grid laid out by row and column, and each block's model and
// parameter values in screen units.
type PresetView struct {
	Format string    `json:"format"`
	Note   string    `json:"note"`
	Name   string    `json:"name"`
	Author string    `json:"author,omitempty"`
	Tempo  uint32    `json:"tempo,omitempty"`
	Volume float32   `json:"volume"`
	Pan    float32   `json:"pan"`
	Tags   []string  `json:"tags,omitempty"`
	Rows   []RowView `json:"rows"`
}

// RowView is one grid row (lane 1..4) with its blocks in column order.
type RowView struct {
	Row    uint32      `json:"row"`
	Blocks []BlockView `json:"blocks"`
}

// BlockView is one block on the grid: its column, model, wire hash and its
// knobs with real (screen) values.
type BlockView struct {
	Column uint32            `json:"column"`
	Model  string            `json:"model"`
	Hash   uint32            `json:"hash"`
	Params map[string]string `json:"params"`
}

// PresetJSON renders a decoded preset as an indented JSON sidecar. It never
// fails on a valid preset: parameter formatting already happened during
// decoding, so any error is a genuine marshal error.
func PresetJSON(cat *Catalog, preset *BinaryPreset) (string, error) {
	view := PresetView{
		Format: presetJSONFormat,
		Note:   presetJSONNote,
		Name:   preset.Name,
		Author: preset.AuthorName,
		Tempo:  preset.Tempo,
		Volume: preset.Volume,
		Pan:    preset.Pan,
		Tags:   preset.Tags,
	}
	for _, c := range preset.Chains {
		row := RowView{Row: c.GetRow(), Blocks: make([]BlockView, 0, len(c.Models))}
		for _, model := range c.Models {
			row.Blocks = append(row.Blocks, BlockView{
				Column: model.GetColumn(),
				Model:  modelName(cat, model),
				Hash:   model.GetHash(),
				Params: blockParamMap(cat, model),
			})
		}
		view.Rows = append(view.Rows, row)
	}
	out, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return "", fmt.Errorf("qc: render preset JSON: %w", err)
	}
	return string(out), nil
}

// blockParamMap returns a model's knobs keyed by name with their screen
// values, falling back to the catalog defaults so the view is self-contained.
func blockParamMap(cat *Catalog, model *Model) map[string]string {
	m, ok := cat.Model(int(model.GetHash()))
	if !ok {
		return nil
	}
	kvs := blockParamKVs(m, model)
	out := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		out[kv.name] = kv.value
	}
	return out
}
