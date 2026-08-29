package rig

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/d-led/guitar-modeler-mcp/internal/assets"
	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
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
	FootSwitch any `json:"FootSwitch"`
	Pedal1     any `json:"Pedal1"`
	Pedal2     any `json:"Pedal2"`
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

	blocks := c.blocks()
	moduleNames := make([]string, 0, len(blocks))
	nodes := make(map[string]*Node, len(blocks)+5)
	seen := make(map[string]int, len(blocks))

	for _, block := range blocks {
		canon := block.Type
		if err := b.validateBlockParams(canon, block.Params); err != nil {
			return nil, err
		}
		var node *Node
		switch canon {
		case "Amp":
			model := strParam(block.Params, "Type")
			if model == "" {
				return nil, fmt.Errorf("amp block requires a \"Type\" parameter with the amp model")
			}
			node = ampNode(model, block.Params)
		case "Cab":
			cab := strParam(block.Params, "CabType")
			if cab == "" {
				return nil, fmt.Errorf("cab block requires a \"CabType\" parameter with the cabinet model")
			}
			mic := strParam(block.Params, "MicType")
			if mic == "" {
				mic = "Dyn 57"
			}
			node = cabNode(cab, mic, block.Params)
		default:
			node, err = buildFXNode(canon, block.Enabled, block.Params)
			if err != nil {
				return nil, err
			}
		}

		// The device names repeated modules with a " N" suffix ("Amp" and
		// "Amp 2", "Cab" and "Cab 2", ...), and the chain slots reference
		// those instance names.
		name := instanceName(canon, seen)
		moduleNames = append(moduleNames, name)
		nodes[name] = node
	}

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

	// Refuse implausible rigs (accidentally very loud or muted) before any
	// file is written, with remediation hints.
	if err := validatePlausible(patch); err != nil {
		return nil, err
	}

	switches, err := resolveFootswitches(spec.Footswitches, moduleNames)
	if err != nil {
		return nil, err
	}
	footSwitch, err := footSwitchFor(b.footSwitch, c.slots(), switches)
	if err != nil {
		return nil, err
	}
	pedal1, err := pedalFor(b.pedal1)
	if err != nil {
		return nil, err
	}
	pedal2, err := pedalFor(b.pedal2)
	if err != nil {
		return nil, err
	}

	content := Content{
		FootSwitch: footSwitch,
		Pedal1:     pedal1,
		Pedal2:     pedal2,
	}
	content.Data.Patch = patch
	content.Info.Version = b.version

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
	id, err := newUUID()
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

func strParam(params map[string]any, key string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
