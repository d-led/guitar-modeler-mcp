// Package mcp implements a minimal Model Context Protocol server over stdio
// (JSON-RPC 2.0, newline-delimited). It supports initialize, ping, tools/list
// and tools/call, which is enough to expose the rig design tools to an agent.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

// Tool describes one callable tool.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handler     func(ctx context.Context, args map[string]any) (string, error)
}

// Server holds registered tools and server metadata.
type Server struct {
	name    string
	version string
	tools   []Tool
	// Log records each tool call by name (not its arguments) so operators can
	// see whether the MCP is being used at all. nil disables logging.
	Log *log.Logger
}

// NewServer creates an MCP server. Tool calls are logged to stderr by default,
// one line per call with the tool's name only; pass a nil Log to silence it.
func NewServer(name, version string) *Server {
	return &Server{
		name:    name,
		version: version,
		Log:     log.New(os.Stderr, "mcp: ", log.LstdFlags),
	}
}

// Register adds a tool to the server.
func (s *Server) Register(t Tool) { s.tools = append(s.tools, t) }

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolResult struct {
	Content []contentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// Run serves requests from r and writes responses to w until EOF or error.
func (s *Server) Run(ctx context.Context, r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	enc := json.NewEncoder(w)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			// Protocol errors with no usable id cannot be correlated; skip.
			continue
		}

		if len(req.ID) == 0 || string(req.ID) == "null" {
			// Notification: no response expected.
			continue
		}

		var id any
		if err := json.Unmarshal(req.ID, &id); err != nil {
			id = string(req.ID)
		}

		result, rpcErr := s.dispatch(ctx, req.Method, req.Params)
		resp := response{JSONRPC: "2.0", ID: id}
		if rpcErr != nil {
			resp.Error = rpcErr
		} else {
			resp.Result = result
		}
		if err := enc.Encode(resp); err != nil {
			return fmt.Errorf("write response: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read request: %w", err)
	}
	return nil
}

func (s *Server) dispatch(ctx context.Context, method string, params json.RawMessage) (any, *rpcError) {
	switch method {
	case "initialize":
		return s.initializeResult(), nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return s.toolsList(), nil
	case "tools/call":
		return s.callTool(ctx, params)
	default:
		return nil, &rpcError{Code: -32601, Message: fmt.Sprintf("method not found: %s", method)}
	}
}

func (s *Server) initializeResult() map[string]any {
	return map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools": map[string]any{"listChanged": false},
		},
		"serverInfo": map[string]any{
			"name":    s.name,
			"version": s.version,
		},
	}
}

func (s *Server) toolsList() map[string]any {
	tools := make([]map[string]any, 0, len(s.tools))
	for _, t := range s.tools {
		tools = append(tools, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
		})
	}
	return map[string]any{"tools": tools}
}

func (s *Server) callTool(ctx context.Context, params json.RawMessage) (any, *rpcError) {
	var call struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, &rpcError{Code: -32602, Message: "invalid params: " + err.Error()}
	}
	for _, t := range s.tools {
		if t.Name != call.Name {
			continue
		}
		if s.Log != nil {
			s.Log.Printf("tool called: %s", call.Name)
		}
		text, err := t.Handler(ctx, call.Arguments)
		if err != nil {
			return toolResult{
				Content: []contentBlock{{Type: "text", Text: err.Error()}},
				IsError: true,
			}, nil
		}
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: text}},
		}, nil
	}
	return nil, &rpcError{Code: -32602, Message: fmt.Sprintf("unknown tool: %s", call.Name)}
}
