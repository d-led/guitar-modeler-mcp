package setlist

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewSetlistOrdersRigs(t *testing.T) {
	s, err := New("Ballad", []Entry{
		{ID: "11111111-1111-4111-8111-111111111111", Name: "Clean"},
		{ID: "22222222-2222-4222-8222-222222222222", Name: "Drive"},
		{ID: "33333333-3333-4333-8333-333333333333", Name: "Solo"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	wantEq(t, "rigs count", len(s.Rigs), 3)
	wantEq(t, "rig_names count", len(s.RigNames), 3)
	wantEq(t, "rig_names[0]", s.RigNames[0], "Clean")
	wantEq(t, "rig_names[2]", s.RigNames[2], "Solo")
	wantEq(t, "rigs[0]", s.Rigs[0], "11111111-1111-4111-8111-111111111111")
	wantEq(t, "author", s.Author, "UserName")
	wantEq(t, "version", s.Version, "1.0.0")
	wantEq(t, "readonly", s.Readonly, false)
	if s.ID == "" || s.CreatedAt == 0 {
		t.Fatal("id/created_at not populated")
	}
}

func wantEq[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func TestNewSetlistMarshalExcludesName(t *testing.T) {
	s, err := New("Ballad", []Entry{{ID: "11111111-1111-4111-8111-111111111111", Name: "Clean"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	data, err := s.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(data), `"name"`) {
		t.Fatalf("setlist JSON must not carry a name field (the file name is the name): %s", data)
	}
	var round struct {
		RigNames []string `json:"rig_names"`
		Rigs     []string `json:"rigs"`
	}
	if err := json.Unmarshal(data, &round); err != nil {
		t.Fatalf("reparse: %v", err)
	}
	if round.RigNames[0] != "Clean" || round.Rigs[0] != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("round-trip mismatch: %+v", round)
	}
}

func TestNewSetlistValidation(t *testing.T) {
	if _, err := New("", []Entry{{ID: "x", Name: "Clean"}}); err == nil {
		t.Fatal("expected an error for an empty name")
	}
	if _, err := New("Ballad", nil); err == nil {
		t.Fatal("expected an error for an empty rig list")
	}
	if _, err := New("Ballad", []Entry{{ID: "", Name: "Clean"}}); err == nil {
		t.Fatal("expected an error for a rig with an empty id")
	}
}
