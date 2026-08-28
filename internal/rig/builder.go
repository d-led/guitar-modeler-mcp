package rig

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/assets"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
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
	footSwitch json.RawMessage
	pedal1     json.RawMessage
	pedal2     json.RawMessage
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
	if err := validateBlocks(b.cat, spec.Blocks); err != nil {
		return nil, err
	}

	moduleNames := make([]string, 0, len(spec.Blocks))
	nodes := make(map[string]*Node, len(spec.Blocks)+5)

	for _, block := range spec.Blocks {
		canon, _ := normalizeBlockName(b.cat, block.Type)
		var node *Node
		var err error

		switch canon {
		case "Amp":
			model := strParam(block.Params, "Type")
			if model == "" {
				return nil, fmt.Errorf("Amp block requires a \"Type\" parameter with the amp model")
			}
			node = ampNode(model, block.Params)
		case "Cab":
			cab := strParam(block.Params, "CabType")
			if cab == "" {
				return nil, fmt.Errorf("Cab block requires a \"CabType\" parameter with the cabinet model")
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

		moduleNames = append(moduleNames, canon)
		nodes[canon] = node
	}

	patch := Patch{
		ChildOrder: append([]string{"Chain", "Rig", "Input", "Output", "Mix"}, moduleNames...),
		Children:   nodes,
	}
	patch.Children["Chain"] = chainNode(moduleNames)
	patch.Children["Rig"] = rigNode(spec.Name, spec.Tempo)
	patch.Children["Input"] = inputNode(spec.InputGain)
	patch.Children["Output"] = outputNode()
	patch.Children["Mix"] = mixNode()

	content := Content{
		FootSwitch: b.footSwitch,
		Pedal1:     b.pedal1,
		Pedal2:     b.pedal2,
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
	id, err := newUUID()
	if err != nil {
		return nil, err
	}

	return &RigFile{
		Author:    author,
		Color:     spec.Color,
		Content:   string(contentJSON),
		CreatedAt: time.Now().Unix(),
		ID:        id,
		Order:     0,
		ProgNum:   0,
		Readonly:  false,
	}, nil
}

// Marshal renders the rig file to indented JSON bytes.
func (f *RigFile) Marshal() ([]byte, error) {
	return json.MarshalIndent(f, "", "  ")
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
