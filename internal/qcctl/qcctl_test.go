package qcctl

import (
	"strings"
	"testing"
)

func TestArgvBuildsAndValidates(t *testing.T) {
	cases := []struct {
		cmd     Command
		want    string
		wantErr bool
	}{
		{Command{Sub: "version"}, "version", false},
		{Command{Sub: "recall", Slot: "28C"}, "recall --slot 28C", false},
		{Command{Sub: "recall", Slot: "28C", Setlist: "/media/p4/Presets"}, "recall --setlist /media/p4/Presets --slot 28C", false},
		{Command{Sub: "scene", Scene: 3}, "scene --index 3", false},
		{Command{Sub: "scene", Scene: 0}, "scene --index 0", false}, // scene A is 0
		{Command{Sub: "dump-preset", Slot: "28C"}, "dump-preset --slot 28C", false},
		{Command{Sub: "recall"}, "", true},           // slot required
		{Command{Sub: "scene", Scene: -1}, "", true}, // index required (-1 = unset)
		{Command{Sub: "format"}, "", true},           // unknown
	}
	for _, c := range cases {
		argv, err := c.cmd.Argv()
		if c.wantErr {
			if err == nil {
				t.Errorf("%+v: expected error, got %v", c.cmd, argv)
			}
			continue
		}
		if err != nil {
			t.Errorf("%+v: %v", c.cmd, err)
			continue
		}
		if got := strings.Join(argv, " "); got != c.want {
			t.Errorf("%+v: argv = %q, want %q", c.cmd, got, c.want)
		}
	}
}

func TestCommandString(t *testing.T) {
	if got := (Command{Sub: "recall", Slot: "28C"}).String(); got != "qcctl --slot 28C" {
		t.Fatalf("String() = %q", got)
	}
}

func TestUnknownCommandRefused(t *testing.T) {
	if _, err := (Command{Sub: "wipe"}).Argv(); err == nil {
		t.Fatal("unknown subcommand accepted")
	}
}
