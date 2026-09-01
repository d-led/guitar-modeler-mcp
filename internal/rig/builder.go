package rig

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/d-led/guitar-modeler-mcp/internal/assets"
	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/fileutil"
)

// RigFile is the outer JSON document of a .rig file.
type RigFile struct {
	Author    string `json:"author"`
	Color     int    `json:"color"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"`
	ID        string `json:"id"`
	Order     int    `json:"order"`
	ProgNum   int    `json:"prog_num"`
	Readonly  bool   `json:"readonly"`
}

// Content is the decoded value of RigFile.Content.
type Content struct {
	FootSwitch json.RawMessage `json:"FootSwitch"`
	Pedal1     json.RawMessage `json:"Pedal1"`
	Pedal2     json.RawMessage `json:"Pedal2"`
	Data       struct {
		Patch Patch `json:"Patch"`
	} `json:"data"`
	Info struct {
		Version string `json:"version"`
	} `json:"info"`
}

// Patch is the signal-chain section of a rig.
type Patch struct {
	ChildOrder []string         `json:"childorder"`
	Children   map[string]*Node `json:"children"`
}

// Decode parses the embedded Content string of a rig file.
func (f *RigFile) Decode() (*Content, error) {
	var c Content
	if err := json.Unmarshal([]byte(f.Content), &c); err != nil {
		return nil, fmt.Errorf("decode rig content: %w", err)
	}
	return &c, nil
}

// Builder produces rig files from Specs, reusing a valid FootSwitch/Pedal
// template captured from the device.
type Builder struct {
	cat *catalog.Catalog
	tmpl
}

type tmpl struct {
	footSwitch []byte
	pedal1     []byte
	pedal2     []byte
	version    string
}

// NewBuilder creates a Builder, loading the embedded rig template.
func NewBuilder(cat *catalog.Catalog) (*Builder, error) {
	var outer struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(assets.TemplateRig(), &outer); err != nil {
		return nil, fmt.Errorf("parse rig template: %w", err)
	}
	var inner struct {
		FootSwitch json.RawMessage `json:"FootSwitch"`
		Pedal1     json.RawMessage `json:"Pedal1"`
		Pedal2     json.RawMessage `json:"Pedal2"`
		Info       struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.Unmarshal([]byte(outer.Content), &inner); err != nil {
		return nil, fmt.Errorf("parse rig template content: %w", err)
	}
	return &Builder{
		cat: cat,
		tmpl: tmpl{
			footSwitch: inner.FootSwitch,
			pedal1:     inner.Pedal1,
			pedal2:     inner.Pedal2,
			version:    inner.Info.Version,
		},
	}, nil
}

// Build renders a Spec into a RigFile.
func (b *Builder) Build(spec Spec) (*RigFile, error) {
	c, err := buildChain(b.cat, spec)
	if err != nil {
		return nil, err
	}

	moduleNames, nodes, err := b.buildNodes(c.blocks())
	if err != nil {
		return nil, err
	}

	patch := b.assemblePatch(spec, c, moduleNames, nodes)

	// Refuse implausible rigs (accidentally very loud or muted) before any
	// file is written, with remediation hints.
	if err := validatePlausible(patch); err != nil {
		return nil, err
	}

	content, err := b.encodeContent(spec, c, patch, moduleNames)
	if err != nil {
		return nil, err
	}
	return b.newRigFile(spec, content)
}

// buildNodes turns each block into a named module node, applying the device's
// " N" suffix for repeated modules and validating each block's parameters.
func (b *Builder) buildNodes(blocks []Block) ([]string, map[string]*Node, error) {
	moduleNames := make([]string, 0, len(blocks))
	nodes := make(map[string]*Node, len(blocks)+5)
	seen := make(map[string]int, len(blocks))

	for _, block := range blocks {
		node, err := b.buildBlockNode(block)
		if err != nil {
			return nil, nil, err
		}
		name := instanceName(block.Type, seen)
		moduleNames = append(moduleNames, name)
		nodes[name] = node
	}
	return moduleNames, nodes, nil
}

// buildBlockNode validates one block's parameters and renders it into a module
// node of the correct kind.
func (b *Builder) buildBlockNode(block Block) (*Node, error) {
	if err := b.validateBlockParams(block.Type, block.Params); err != nil {
		return nil, err
	}
	switch block.Type {
	case "Amp":
		model := strParam(block.Params, "Type")
		if model == "" {
			return nil, fmt.Errorf("amp block requires a \"Type\" parameter with the amp model")
		}
		return ampNode(model, block.Params), nil
	case "Cab":
		cab := strParam(block.Params, "CabType")
		if cab == "" {
			return nil, fmt.Errorf("cab block requires a \"CabType\" parameter with the cabinet model")
		}
		mic := strParam(block.Params, "MicType")
		if mic == "" {
			mic = "Dyn 57"
		}
		return cabNode(cab, mic, block.Params), nil
	case "IR", "IR (1024)":
		return irNode(block.Enabled, block.Params), nil
	default:
		return buildFXNode(block.Type, block.Enabled, block.Params)
	}
}

// assemblePatch builds the Patch section: the fixed Chain/Rig/Input/Output/Mix
// bookkeeping nodes plus the module nodes, in device order.
func (b *Builder) assemblePatch(spec Spec, c chain, moduleNames []string, nodes map[string]*Node) Patch {
	pathMix := pathMixFor(spec)
	patch := Patch{
		ChildOrder: append([]string{"Chain", "Rig", "Input", "Output", "Mix"}, moduleNames...),
		Children:   nodes,
	}
	patch.Children["Chain"] = chainNode(c.routing, c.slots(), pathMix)
	patch.Children["Rig"] = rigNode(spec.Name, spec.Tempo)
	patch.Children["Input"] = inputNode(spec.InputGain)
	patch.Children["Output"] = outputNode(spec.OutputVolume)
	patch.Children["Mix"] = mixNode(pathMix)
	return patch
}

// encodeContent wires the footswitch and pedal templates to the new chain and
// assembles the inner Content document.
func (b *Builder) encodeContent(spec Spec, c chain, patch Patch, moduleNames []string) (Content, error) {
	switches, err := resolveFootswitches(spec.Footswitches, moduleNames)
	if err != nil {
		return Content{}, err
	}
	footSwitch, err := footSwitchFor(b.footSwitch, c.slots(), switches)
	if err != nil {
		return Content{}, err
	}
	pedals, err := resolvePedals(spec.Pedals, moduleNames)
	if err != nil {
		return Content{}, err
	}
	pedal1, err := pedalFor(b.pedal1, pedalAt(pedals, 0))
	if err != nil {
		return Content{}, err
	}
	pedal2, err := pedalFor(b.pedal2, pedalAt(pedals, 1))
	if err != nil {
		return Content{}, err
	}

	content := Content{}
	if content.FootSwitch, err = json.Marshal(footSwitch); err != nil {
		return Content{}, fmt.Errorf("encode footswitch section: %w", err)
	}
	if content.Pedal1, err = json.Marshal(pedal1); err != nil {
		return Content{}, fmt.Errorf("encode pedal1 section: %w", err)
	}
	if content.Pedal2, err = json.Marshal(pedal2); err != nil {
		return Content{}, fmt.Errorf("encode pedal2 section: %w", err)
	}
	content.Data.Patch = patch
	content.Info.Version = b.version
	return content, nil
}

// newRigFile wraps the encoded content in the outer RigFile envelope with the
// spec's metadata and a fresh id.
func (b *Builder) newRigFile(spec Spec, content Content) (*RigFile, error) {
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("encode rig content: %w", err)
	}

	author := spec.Author
	if author == "" {
		author = "UserName"
	}
	// The device's rig colour is an integer 1..9 (0 is not accepted). A zero
	// spec colour falls back to the factory default.
	color := spec.Color
	if color == 0 {
		color = 4
	}
	if color < 1 || color > 9 {
		return nil, fmt.Errorf("rig colour must be 1..9, got %d", spec.Color)
	}
	id, err := fileutil.NewUUID()
	if err != nil {
		return nil, err
	}

	return &RigFile{
		Author:    author,
		Color:     color,
		Content:   string(contentJSON),
		CreatedAt: time.Now().Unix(),
		ID:        id,
		Order:     0,
		ProgNum:   -1,
		Readonly:  false,
	}, nil
}

// Marshal renders the rig file as compact, single-line JSON, exactly as the
// device writes it (no indentation and no trailing newline).
func (f *RigFile) Marshal() ([]byte, error) {
	return json.Marshal(f)
}

// SetContent re-encodes a decoded (and possibly modified) content model back
// into the rig's inner JSON, completing a Load → Change → Save round-trip. The
// FootSwitch/Pedal sections are held as raw JSON, so untouched sections survive
// byte-for-byte.
func (f *RigFile) SetContent(c *Content) error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode rig content: %w", err)
	}
	f.Content = string(data)
	return nil
}

func strParam(params map[string]any, key string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
