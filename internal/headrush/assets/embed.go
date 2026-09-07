// Package assets embeds the device data captured from the HeadRush Gigboard
// backup: every effect module's factory block definitions and a known-good rig
// template. Embedding the data keeps the MCP server self-contained.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed data/blocks
var blocksFS embed.FS

//go:embed data/template.rig
var templateRig []byte

// Blocks returns the embedded block definitions file system rooted at
// data/blocks. Each sub-directory is named after the module type (uppercase)
// and contains one or more .block files.
func Blocks() fs.FS {
	sub, err := fs.Sub(blocksFS, "data/blocks")
	if err != nil {
		// Unreachable: the directory is embedded at compile time.
		panic(err)
	}
	return sub
}

// TemplateRig returns the raw bytes of the embedded rig template, which
// provides valid FootSwitch/Pedal sections and the schema version.
func TemplateRig() []byte {
	return templateRig
}
