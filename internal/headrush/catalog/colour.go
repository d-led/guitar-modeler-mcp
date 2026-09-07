package catalog

import "strings"

// moduleColours are the per-slot colour tags the device accepts (the Colour
// item every module carries, shown as a tag in the editor grid). Each chain
// slot can have its own colour, independent of the module's model.
var moduleColours = []string{
	"Blue", "Yellow", "Green", "Purple", "Red", "Dark Green", "Orange", "Light Blue", "Pink",
}

// Colours returns the module slot colours the device accepts, in editor order.
func Colours() []string { return moduleColours }

// ColourValid reports whether c is a module slot colour the device accepts.
// Matching is exact: the device stores the tags as written ("Dark Green").
func ColourValid(c string) bool {
	for _, col := range moduleColours {
		if col == c {
			return true
		}
	}
	return false
}

// ColourList joins the accepted colours for an error message or a schema
// description, e.g. `Blue, Yellow, ...`.
func ColourList() string { return strings.Join(moduleColours, ", ") }
