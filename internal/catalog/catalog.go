package catalog

import "strings"

// Catalog aggregates the device model data and provides lookups.
type Catalog struct {
	amps map[string]Amp
	cabs map[string]Cab
	mics map[string]Mic
	fx   map[string]FX
}

// New builds a Catalog from the embedded data.
func New() *Catalog {
	c := &Catalog{
		amps: make(map[string]Amp, len(amps)),
		cabs: make(map[string]Cab, len(cabs)),
		mics: make(map[string]Mic, len(mics)),
		fx:   make(map[string]FX, len(fx)),
	}
	for _, a := range amps {
		c.amps[strings.ToLower(a.Model)] = a
	}
	for _, cb := range cabs {
		c.cabs[strings.ToLower(cb.Model)] = cb
	}
	for _, m := range mics {
		c.mics[strings.ToLower(m.Model)] = m
	}
	for _, f := range fx {
		c.fx[strings.ToLower(f.Name)] = f
	}
	return c
}

// Amps returns all amp models.
func (c *Catalog) Amps() []Amp { return amps }

// Cabs returns all cabinet models.
func (c *Catalog) Cabs() []Cab { return cabs }

// Mics returns all microphone models.
func (c *Catalog) Mics() []Mic { return mics }

// FX returns all effect modules.
func (c *Catalog) FX() []FX { return fx }

// Amp returns the amp with the given device model name (case-insensitive).
func (c *Catalog) Amp(model string) (Amp, bool) {
	a, ok := c.amps[strings.ToLower(model)]
	return a, ok
}

// Cab returns the cab with the given device model name (case-insensitive).
func (c *Catalog) Cab(model string) (Cab, bool) {
	cb, ok := c.cabs[strings.ToLower(model)]
	return cb, ok
}

// Mic returns the mic with the given device model name (case-insensitive).
func (c *Catalog) Mic(model string) (Mic, bool) {
	m, ok := c.mics[strings.ToLower(model)]
	return m, ok
}

// FX returns the effect with the given device module name (case-insensitive).
func (c *Catalog) FXByName(name string) (FX, bool) {
	f, ok := c.fx[strings.ToLower(name)]
	return f, ok
}

// AmpsMatching filters amps by a case-insensitive substring query across all
// searchable fields. An empty query returns the full list.
func (c *Catalog) AmpsMatching(query string) []Amp {
	if strings.TrimSpace(query) == "" {
		return amps
	}
	q := strings.ToLower(query)
	var out []Amp
	for _, a := range amps {
		hay := strings.ToLower(a.Model + " " + a.Brand + " " + a.RealModel + " " + a.Channel + " " + a.Wattage + " " + strings.Join(a.Style, " ") + " " + a.Description)
		if strings.Contains(hay, q) {
			out = append(out, a)
		}
	}
	return out
}

// CabsMatching filters cabs by a case-insensitive substring query.
func (c *Catalog) CabsMatching(query string) []Cab {
	if strings.TrimSpace(query) == "" {
		return cabs
	}
	q := strings.ToLower(query)
	var out []Cab
	for _, cb := range cabs {
		hay := strings.ToLower(cb.Model + " " + cb.Speakers + " " + cb.SpeakersRef + " " + cb.Description)
		if strings.Contains(hay, q) {
			out = append(out, cb)
		}
	}
	return out
}

// MicsMatching filters mics by a case-insensitive substring query.
func (c *Catalog) MicsMatching(query string) []Mic {
	if strings.TrimSpace(query) == "" {
		return mics
	}
	q := strings.ToLower(query)
	var out []Mic
	for _, m := range mics {
		hay := strings.ToLower(m.Model + " " + m.Kind + " " + m.RealModel + " " + m.Description)
		if strings.Contains(hay, q) {
			out = append(out, m)
		}
	}
	return out
}
