package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"testing"
)

func TestRunHandlesInitializeListAndCall(t *testing.T) {
	s := NewServer("test-server", "1.2.3")
	s.Register(Tool{
		Name:        "echo",
		Description: "echoes a name",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}}},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return "hi " + args["name"].(string), nil
		},
	})

	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"name":"bob"}}}
`
	var out strings.Builder
	if err := s.Run(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatalf("Run: %v", err)
	}

	lines := nonEmptyLines(out.String())
	if len(lines) != 3 {
		t.Fatalf("got %d responses, want 3 (notification must not be answered):\n%s", len(lines), out.String())
	}

	var initResp struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
			ServerInfo      struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("initialize response: %v", err)
	}
	if initResp.Result.ServerInfo.Name != "test-server" || initResp.Result.ServerInfo.Version != "1.2.3" {
		t.Fatalf("serverInfo = %+v", initResp.Result.ServerInfo)
	}
	if initResp.Result.ProtocolVersion == "" {
		t.Fatal("missing protocolVersion")
	}

	var listResp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &listResp); err != nil {
		t.Fatalf("tools/list response: %v", err)
	}
	if len(listResp.Result.Tools) != 1 || listResp.Result.Tools[0].Name != "echo" {
		t.Fatalf("tools = %+v", listResp.Result.Tools)
	}

	var callResp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[2]), &callResp); err != nil {
		t.Fatalf("tools/call response: %v", err)
	}
	if callResp.Result.Content[0].Text != "hi bob" {
		t.Fatalf("content = %+v", callResp.Result.Content)
	}
	if callResp.Result.IsError {
		t.Fatal("unexpected isError")
	}
}

func TestRunLogsToolCallByNameOnly(t *testing.T) {
	var logs bytes.Buffer
	s := NewServer("test", "1")
	s.Log = log.New(&logs, "mcp: ", 0) // no timestamps, so assertions are exact
	s.Register(Tool{
		Name:        "echo",
		Description: "echoes",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{"secret": map[string]any{"type": "string"}}},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return "ok", nil
		},
	})

	var out strings.Builder
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{"secret":"do-not-log-this"}}}
`
	if err := s.Run(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := logs.String()
	if got != "mcp: tool called: echo\n" {
		t.Fatalf("log output = %q, want exactly the tool name", got)
	}
	if strings.Contains(got, "do-not-log-this") {
		t.Fatal("arguments leaked into the log")
	}
}

func TestRunReportsToolErrorAsResult(t *testing.T) {
	s := NewServer("test", "1")
	s.Register(Tool{
		Name:        "fail",
		Description: "always fails",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(_ context.Context, _ map[string]any) (string, error) {
			return "", errors.New("boom")
		},
	})

	var out strings.Builder
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"fail","arguments":{}}}
`
	if err := s.Run(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	var resp struct {
		Result struct {
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &resp); err != nil {
		t.Fatalf("response: %v", err)
	}
	if !resp.Result.IsError {
		t.Fatal("expected isError true")
	}
}

func nonEmptyLines(s string) []string {
	var out []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			out = append(out, line)
		}
	}
	return out
}
