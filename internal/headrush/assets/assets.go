package assets

import (
	"io/fs"
	"path"
	"sort"
	"strings"
)

// ModuleTypes returns the uppercase module type names (the directory names
// under data/blocks), sorted alphabetically. These map 1:1 to the device
// module display names via strings.ToUpper.
func ModuleTypes() []string {
	entries, err := fs.ReadDir(Blocks(), ".")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// Presets lists the named presets (block files) available for a module type.
// The returned names have the ".block" extension stripped.
func Presets(moduleType string) ([]string, error) {
	entries, err := fs.ReadDir(Blocks(), moduleType)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".block") {
			out = append(out, strings.TrimSuffix(e.Name(), ".block"))
		}
	}
	sort.Strings(out)
	return out, nil
}

// ReadBlock returns the raw JSON bytes of a single preset block file.
func ReadBlock(moduleType, preset string) ([]byte, error) {
	return fs.ReadFile(Blocks(), path.Join(moduleType, preset+".block"))
}

// DefaultBlock returns the raw JSON bytes of the factory default block for a
// module type. It prefers files whose name starts with "+DEFAULT"; if none
// exists it falls back to the first preset alphabetically.
func DefaultBlock(moduleType string) ([]byte, error) {
	names, err := Presets(moduleType)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fs.ErrNotExist
	}
	for _, n := range names {
		if strings.HasPrefix(n, "+DEFAULT") {
			return ReadBlock(moduleType, n)
		}
	}
	return ReadBlock(moduleType, names[0])
}
