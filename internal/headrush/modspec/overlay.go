package modspec

// This file applies corrections to the extracted editor spec, derived from
// cross-checking against the device backup (the ground truth). The raw
// params.json stays regenerable; these fixes are explicit and reviewed.

// noteValues is the device's tempo-sync vocabulary, copied from the delay
// modules whose editor spec already models sync as a dropdown.
var noteValues = []string{
	"1/128", "1/64", "1/32T", "1/32", "1/16T", "1/16", "1/8T", "1/8",
	"1/4T", "3/16", "1/4", "1/2T", "3/8", "1/2", "5/8", "3/4", "Bar",
}

func fp(v float64) *float64 { return &v }

// applyOverlay mutates the loaded module specs with the corrections below.
func applyOverlay() {
	fixTypos()
	markSyncParams()
	addAmpTremSpeed2()
	fixEditorRangeBugs()
	addMissingSetValues()
}

// fixTypos renames parameter keys that the editor spec misspells.
func fixTypos() {
	renameParam("FilterThreshhold", "FilterThreshold")
}

func renameParam(from, to string) {
	for _, params := range modules {
		if p, ok := params[from]; ok {
			delete(params, from)
			params[to] = p
		}
	}
}

// markSyncParams converts time/rate params to the "sync" kind in every module
// that has a Sync (or TremSync) toggle: those params accept either a number or
// a tempo note value, which the editor's range-only spec does not capture.
func markSyncParams() {
	for modName, params := range modules {
		hasSync := params["Sync"].Kind == "toggle" || (modName == "Amp" && params["TremSync"].Kind == "toggle")
		if !hasSync {
			continue
		}
		for key, p := range params {
			if p.Kind != "range" || !isTimeParam(key) {
				continue
			}
			p.Kind = "sync"
			p.Values = noteValues
			params[key] = p
		}
	}
}

func isTimeParam(key string) bool {
	switch key {
	case "Rate", "LFORate", "Delay", "Delay1", "Delay2", "TremSpeed", "TremSpeed2":
		return true
	}
	return false
}

// addAmpTremSpeed2 adds the amp's second tremolo-speed parameter, which the
// editor spec omits.
func addAmpTremSpeed2() {
	amp, ok := modules["Amp"]
	if !ok {
		return
	}
	if _, exists := amp["TremSpeed2"]; exists {
		return
	}
	amp["TremSpeed2"] = Param{Kind: "sync", Label: "Trem Speed", Min: fp(0.25), Max: fp(20), Step: fp(0.01), Unit: " Hz", Values: noteValues}
}

// fixEditorRangeBugs relaxes ranges that the editor got wrong, using the
// minimum/maximum values actually observed in the device backup.
func fixEditorRangeBugs() {
	fixRange("8-Bit Crush", "AntiAliasing", nil, fp(100)) // editor said max 1, device uses 0-100
	fixRange("Flanger", "Feedback", fp(-100), nil)        // bipolar feedback
	fixRange("Flanger", "Rate", fp(0.01), nil)            // editor said 0.1, device goes down to 0.01
	fixRange("Tron Filter", "Reso", nil, fp(100))         // editor said 10, device observed 70
}

func fixRange(module, key string, lo, hi *float64) {
	m, ok := modules[module]
	if !ok {
		return
	}
	p, ok := m[key]
	if !ok || p.Kind != "range" {
		return
	}
	if lo != nil {
		p.Min = lo
	}
	if hi != nil {
		p.Max = hi
	}
	m[key] = p
}

// addMissingSetValues appends enum options observed in the backup but absent
// from the editor spec.
func addMissingSetValues() {
	addSetValue("AIR Reverb", "Mode", "NonLinear")
}

func addSetValue(module, key, value string) {
	m, ok := modules[module]
	if !ok {
		return
	}
	p, ok := m[key]
	if !ok || p.Kind != "set" {
		return
	}
	for _, v := range p.Values {
		if v == value {
			return
		}
	}
	p.Values = append(p.Values, value)
	m[key] = p
}
