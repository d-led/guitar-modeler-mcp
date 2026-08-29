package mooer

import "strings"

// models is the registry of supported Mooer devices, in display order.
var models = []Model{ge150pro(), ge200(), ge150(), ge100pro()}

// Models returns the supported Mooer devices.
func Models() []Model {
	return models
}

// ModelByName returns the model with the given identifier, matching
// case-insensitively against both the stable name and the display name.
func ModelByName(name string) (Model, bool) {
	q := strings.ToLower(strings.TrimSpace(name))
	for _, m := range models {
		if strings.ToLower(m.Name) == q || strings.ToLower(m.Display) == q {
			return m, true
		}
	}
	return Model{}, false
}
