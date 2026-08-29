package mooer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMORoundTrip(t *testing.T) {
	want := fullPreset()

	got, err := UnmarshalMO(MarshalMO(want))
	if err != nil {
		t.Fatalf("UnmarshalMO: %v", err)
	}
	if got != want {
		t.Fatalf("MO round trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestMOCarriesZeroedHeaderAndPadding(t *testing.T) {
	raw := MarshalMO(New())
	for _, i := range []int{0, 1, MOPresetOffset - 1} {
		if raw[i] != 0 {
			t.Fatalf("header byte %d = %d, want 0", i, raw[i])
		}
	}
	for _, i := range []int{MOPresetOffset + PresetSize, MOFileSize - 1} {
		if raw[i] != 0 {
			t.Fatalf("padding byte %d = %d, want 0", i, raw[i])
		}
	}
}

func TestWriteAndReadMOFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tone.mo")

	want := fullPreset()
	if err := WriteMOFile(path, want); err != nil {
		t.Fatalf("WriteMOFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() != MOFileSize {
		t.Fatalf("file size = %d, want %d", info.Size(), MOFileSize)
	}

	got, err := ReadMOFile(path)
	if err != nil {
		t.Fatalf("ReadMOFile: %v", err)
	}
	if got != want {
		t.Fatal("file round trip mismatch")
	}
}

func TestUnmarshalMORejectsShortFile(t *testing.T) {
	if _, err := UnmarshalMO(make([]byte, MOPresetOffset+PresetSize-1)); err == nil {
		t.Fatal("expected an error for a short .mo file")
	}
}
