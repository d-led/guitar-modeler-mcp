package waza

import (
	"fmt"
	"html"
	"strings"
)

// Spec is a tone to dial in on the Waza Air: the selected amp, effects and
// spatial settings. Empty fields are left off the setup card; a zero Gain,
// Volume or DelayTime means "leave to BOSS TONE STUDIO".
type Spec struct {
	Name         string
	Amp          string
	Booster      string
	Mod          string
	FX           string
	Delay        string
	Reverb       string
	CabResonance string
	Ambience     string
	Position     string
	Mode         string
	Gain         int
	Volume       int
	DelayTime    int
}

// Resolve canonicalises the spec's selections to on-device names. Each field
// matches an exact model name first, then a substring of the "inspired by"
// description; an empty field is left empty.
func (d Device) Resolve(s Spec) (Spec, error) {
	var err error
	if s.Amp, err = resolve(d.Amps, s.Amp, "amp"); err != nil {
		return s, err
	}
	if s.Booster, err = resolve(d.Boosters, s.Booster, "booster"); err != nil {
		return s, err
	}
	if s.Mod, err = resolve(d.ModFX, s.Mod, "mod"); err != nil {
		return s, err
	}
	if s.FX, err = resolve(d.ModFX, s.FX, "fx"); err != nil {
		return s, err
	}
	if s.Delay, err = resolve(d.Delays, s.Delay, "delay"); err != nil {
		return s, err
	}
	if s.Reverb, err = resolve(d.Reverbs, s.Reverb, "reverb"); err != nil {
		return s, err
	}
	if s.CabResonance, err = resolveString(d.CabResonance, s.CabResonance); err != nil {
		return s, err
	}
	if s.Ambience, err = resolveString(d.Ambience, s.Ambience); err != nil {
		return s, err
	}
	if s.Position, err = resolveString(d.Position, s.Position); err != nil {
		return s, err
	}
	if s.Mode, err = resolveString(d.Mode, s.Mode); err != nil {
		return s, err
	}
	return s, nil
}

func resolve(items []Item, query, label string) (string, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "", nil
	}
	for _, it := range items {
		if strings.EqualFold(it.Name, q) {
			return it.Name, nil
		}
	}
	for _, it := range items {
		if it.InspiredBy != "" && strings.Contains(strings.ToLower(it.InspiredBy), strings.ToLower(q)) {
			return it.Name, nil
		}
	}
	return "", fmt.Errorf("no %s matches %q", label, q)
}

func resolveString(options []string, query string) (string, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "", nil
	}
	for _, o := range options {
		if strings.EqualFold(o, q) {
			return o, nil
		}
	}
	return "", fmt.Errorf("unknown value %q (valid: %s)", q, strings.Join(options, ", "))
}

func (d Device) inspired(name string, items []Item) string {
	for _, it := range items {
		if strings.EqualFold(it.Name, name) {
			return it.InspiredBy
		}
	}
	return ""
}

// SetupCardHTML renders a printable setup card for a resolved Spec. It is the
// only output for the card-only Waza Air.
func (d Device) SetupCardHTML(s Spec) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\">")
	fmt.Fprintf(&b, "<title>%s — %s</title>", html.EscapeString(s.Name), html.EscapeString(d.Display))
	b.WriteString(`<style>
body{font-family:system-ui,-apple-system,sans-serif;max-width:720px;margin:2rem auto;padding:0 1rem;color:#1a1a1a}
h1{margin-bottom:.25rem}h2{font-size:1rem;color:#555;margin-top:0}
table{width:100%;border-collapse:collapse;margin-bottom:1rem}
td,th{border-bottom:1px solid #e2e2e2;padding:.45rem .5rem;text-align:left;vertical-align:top}
.module{font-weight:600;white-space:nowrap}
.effect{font-weight:600}.inspired{color:#666;font-size:.85em}
</style></head><body>`)
	fmt.Fprintf(&b, "<h1>%s</h1><h2>%s — setup card</h2>", html.EscapeString(s.Name), html.EscapeString(d.Display))

	for _, module := range d.Chain {
		var effect, inspired string
		switch module {
		case "BOOSTER":
			effect, inspired = s.Booster, d.inspired(s.Booster, d.Boosters)
		case "AMP":
			effect, inspired = s.Amp, d.inspired(s.Amp, d.Amps)
		case "MOD":
			effect, inspired = s.Mod, d.inspired(s.Mod, d.ModFX)
		case "FX":
			effect, inspired = s.FX, d.inspired(s.FX, d.ModFX)
		case "DELAY":
			effect, inspired = s.Delay, d.inspired(s.Delay, d.Delays)
		case "REVERB":
			effect, inspired = s.Reverb, d.inspired(s.Reverb, d.Reverbs)
		}
		writeModule(&b, module, effect, inspired)
	}

	writeSetting(&b, "CABINET RESONANCE", s.CabResonance)
	writeSetting(&b, "AMBIENCE", s.Ambience)
	writeSetting(&b, "POSITION", s.Position)
	writeSetting(&b, "MODE", s.Mode)

	if s.Gain > 0 {
		writeSetting(&b, "AMP GAIN", fmt.Sprintf("%d", s.Gain))
	}
	if s.Volume > 0 {
		writeSetting(&b, "AMP VOLUME", fmt.Sprintf("%d", s.Volume))
	}
	if s.DelayTime > 0 {
		writeSetting(&b, "DELAY TIME", fmt.Sprintf("%d ms", s.DelayTime))
	}

	b.WriteString("<p class=\"inspired\">Effect parameter values are set in BOSS TONE STUDIO.</p>")
	b.WriteString("</body></html>")
	return b.String()
}

func writeModule(b *strings.Builder, module, effect, inspired string) {
	if effect == "" {
		effect = "OFF"
	}
	fmt.Fprintf(b, "<table><tr><td class=\"module\">%s</td><td class=\"effect\">%s</td></tr>", html.EscapeString(module), html.EscapeString(effect))
	if inspired != "" {
		fmt.Fprintf(b, "<tr><td></td><td><span class=\"inspired\">based on %s</span></td></tr>", html.EscapeString(inspired))
	}
	b.WriteString("</table>")
}

func writeSetting(b *strings.Builder, name, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(b, "<table><tr><td class=\"module\">%s</td><td class=\"effect\">%s</td></tr></table>", html.EscapeString(name), html.EscapeString(value))
}
