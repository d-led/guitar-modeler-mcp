package qc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Grid port constants, from the device's own protocol (pyquadcortex enums).
const (
	InputInput1    uint32 = 1  // "Input 1"
	InputInput2    uint32 = 2  // "Input 2"
	OutputMultiple uint32 = 19 // the Multi-Out destination factory presets use
	laneOutputHash        = 23000
)

// BlockSpec is one block to place on the grid, in signal-chain order.
type BlockSpec struct {
	// Model resolves the block: its on-device name, or a "based on"
	// description ("JCM800", "TS808", "Vintage 30 4x12").
	Model string
	// Params are named knob settings in the block's own units: screen units
	// for continuous knobs (GAIN 5 on a 0..10 control, LEVEL -3 on a dB knob),
	// and the option index for list parameters (MODE 1). Values are normalised
	// through the catalog's skew, so 5 on a linear 0..10 knob lands at wire 0.5.
	Params map[string]float64
	// EncodedParams are named knobs given directly on the device's 0..1 line.
	// Use these for the few parameters whose bounds nobody has measured.
	EncodedParams map[string]float64
}

// DesignSpec is a serial signal chain to render as a preset: the blocks are
// laid out left-to-right on row 0, into Input 1 and out of the Multi-Out.
type DesignSpec struct {
	Name   string
	Author string
	// Volume is the preset output level; the device's factory presets store 1.0.
	Volume float64
	// Blocks are the amp/cab/effect chain, in order (drives first, then amp,
	// then cab, then time and ambience effects — the caller decides).
	Blocks []BlockSpec
}

// BuildPreset renders a DesignSpec into a BinaryPreset with one serial chain
// on row 0. It resolves each block against the catalog and validates every
// parameter value, so an invalid preset is refused rather than written.
func BuildPreset(cat *Catalog, spec DesignSpec) (*BinaryPreset, error) {
	if strings.TrimSpace(spec.Name) == "" {
		return nil, fmt.Errorf("a preset name is required")
	}
	preset := &BinaryPreset{
		Name:   spec.Name,
		Date:   time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Volume: float32(spec.Volume),
		Pan:    0.5, // centre
		Chains: []*Chain{{
			XInPortid:  &Chain_InPortid{InPortid: InputInput1},
			XOutPortid: &Chain_OutPortid{OutPortid: OutputMultiple},
			XRow:       &Chain_Row{Row: 0},
		}},
	}
	if preset.Volume == 0 {
		preset.Volume = 1.0 // unity, matching factory presets
	}
	if spec.Author != "" {
		preset.AuthorName = spec.Author
	}

	chain := preset.Chains[0]
	for col, block := range spec.Blocks {
		m, err := cat.resolveBlock(block.Model)
		if err != nil {
			return nil, err
		}
		model := &Model{
			XHash:   &Model_Hash{Hash: uint32(m.ID)},
			XColumn: &Model_Column{Column: uint32(col)},
		}
		if err := applyParams(m, block, model); err != nil {
			return nil, fmt.Errorf("%s: %w", m.Name, err)
		}
		chain.Models = append(chain.Models, model)
	}

	// A lane output control is what factory presets carry at the end of a row.
	chain.OutputControl = []*Model{{XHash: &Model_Hash{Hash: laneOutputHash}}}
	return preset, nil
}

// resolveBlock finds a model by name or "based on" description.
func (c *Catalog) resolveBlock(query string) (*ModelSpec, error) {
	if m, ok := c.Find(query); ok {
		return m, nil
	}
	return nil, fmt.Errorf("no Quad Cortex model matches %q", query)
}

// applyParams turns a block's named settings into wire-encoded Param entries,
// in the model's parameter order.
func applyParams(m *ModelSpec, block BlockSpec, model *Model) error {
	seen := map[int]bool{}
	appendParam := func(index int, wire float64) {
		model.Params = append(model.Params, &Param{
			XIndex:      &Param_Index{Index: uint32(index)},
			ParamValues: []*ParamValue{{Value: &ParamValue_FloatValue{FloatValue: float32(wire)}}},
		})
	}

	apply := func(name string, value any) error {
		spec, index, ok := m.ResolveParam(name)
		if !ok {
			return fmt.Errorf("%s: unknown parameter %q (available: %s)",
				m.Name, name, strings.Join(m.KnobNames(), ", "))
		}
		if seen[index] {
			return fmt.Errorf("parameter %q set twice", name)
		}
		seen[index] = true

		wire, err := encodeValue(spec, value)
		if err != nil {
			return err
		}
		appendParam(index, wire)
		return nil
	}

	for name, v := range block.Params {
		if err := apply(name, v); err != nil {
			return err
		}
	}
	for name, v := range block.EncodedParams {
		_, index, ok := m.ResolveParam(name)
		if !ok {
			return fmt.Errorf("%s: unknown parameter %q (available: %s)",
				m.Name, name, strings.Join(m.KnobNames(), ", "))
		}
		if seen[index] {
			return fmt.Errorf("parameter %q set twice", name)
		}
		seen[index] = true
		if v < 0 || v > 1 {
			return fmt.Errorf("%s: an encoded value runs 0..1, got %g", name, v)
		}
		appendParam(index, v)
	}
	return nil
}

// encodeValue converts one setting to the wire's 0..1. A list parameter takes
// an option index or option name; a continuous knob takes a screen value that
// is normalised through the catalog's skew.
func encodeValue(spec ParamSpec, value any) (float64, error) {
	if spec.padding {
		return 0, fmt.Errorf("%s is a wire placeholder, not a knob", spec.Name)
	}
	if spec.Steps > 0 {
		var idx int
		switch v := value.(type) {
		case int:
			idx = v
		case float64:
			idx = int(v)
		case string:
			n, err := spec.OptionName(v)
			if err != nil {
				return 0, err
			}
			idx = n
		default:
			return 0, fmt.Errorf("%s: expected an option index or name, got %T", spec.Name, value)
		}
		return spec.OptionToValue(idx)
	}
	if spec.unmeasured {
		return 0, fmt.Errorf("%s: bounds are unmeasured; put it in encoded_params as a 0..1 value", spec.Name)
	}
	real, ok := value.(float64)
	if !ok {
		return 0, fmt.Errorf("%s: expected a number, got %T", spec.Name, value)
	}
	return spec.Normalize(real)
}

// WritePresetWithCard renders and encrypts a preset for the given serial and
// writes it to <outputDir>/<name>.pb together with a printable HTML setup card
// <outputDir>/<name>.html and a human-readable JSON view
// <outputDir>/<name>.json. It returns all three paths.
func WritePresetWithCard(serial string, spec DesignSpec, outputDir string) (pbPath, cardPath, jsonPath string, err error) {
	cat, err := defaultCatalog()
	if err != nil {
		return "", "", "", err
	}
	preset, err := BuildPreset(cat, spec)
	if err != nil {
		return "", "", "", err
	}
	data, err := EncodePreset(serial, preset)
	if err != nil {
		return "", "", "", err
	}
	stem := sanitizeName(spec.Name)
	pbPath = filepath.Join(outputDir, stem+".pb")
	if err := os.WriteFile(pbPath, data, 0o644); err != nil {
		return "", "", "", fmt.Errorf("write preset: %w", err)
	}
	cardPath = filepath.Join(outputDir, stem+".html")
	if err := os.WriteFile(cardPath, []byte(SetupCardHTML(cat, preset)), 0o644); err != nil {
		return "", "", "", fmt.Errorf("write setup card: %w", err)
	}
	view, err := PresetJSON(cat, preset)
	if err != nil {
		return "", "", "", err
	}
	jsonPath = filepath.Join(outputDir, stem+".json")
	if err := os.WriteFile(jsonPath, []byte(view), 0o644); err != nil {
		return "", "", "", fmt.Errorf("write preset JSON view: %w", err)
	}
	return pbPath, cardPath, jsonPath, nil
}

func sanitizeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte(' ')
		}
	}
	return strings.TrimSpace(strings.Join(strings.Fields(b.String()), " "))
}
