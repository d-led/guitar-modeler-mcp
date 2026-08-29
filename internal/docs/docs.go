// Package docs embeds the agent-facing guide that the MCP server can hand back
// to a model, so the domain knowledge (chain topology, routing constraints,
// workflow) lives in the binary rather than only in the README.
package docs

import _ "embed"

//go:embed agent-guide.md
var agentGuide string

// Guide returns the embedded agent guide as markdown text.
func Guide() string {
	return agentGuide
}
