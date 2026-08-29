package qc

import (
	_ "embed"
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
)

// modelRepoXML is the Quad Cortex block catalog (ModelRepo.xml), the single
// source of truth for model names, wire hashes (ids), parameter names and
// parameter scales. It is transcribed from the device's own catalog via the
// OpenCortex project.
//
//go:embed modelrepo.xml
var modelRepoXML []byte

// firmwareConstants resolves the symbolic min/max bounds the catalog names but
// does not spell out as numbers. Each value's provenance is recorded in the
// pyquadcortex project (units.FIRMWARE_CONSTANTS); we carry only the numbers.
//
// MIN_INPUT_TRIM / MAX_INPUT_TRIM are deliberately absent: the catalog names
// them (NC_Recorder OUT LEVEL) but nobody has measured them, so a conversion
// must refuse rather than guess.
var firmwareConstants = map[string]float64{
	"MIN_CABSIM_DB":          -40.0,
	"MAX_CABSIM_DB":          6.0,
	"MIN_EQ_DB":              -12.0,
	"MAX_EQ_DB":              12.0,
	"MIN_MIXER_DB":           -40.0,
	"MAX_MIXER_DB":           12.0,
	"MIN_FXLOOP_OUT_GAIN_DB": -40.0,
	"MAX_FXLOOP_OUT_GAIN_DB": 0.0,
	"MIN_FXLOOP_IN_GAIN_DB":  -40.0,
	"MAX_FXLOOP_IN_GAIN_DB":  12.0,
	"MIN_TEMPO":              40.0,
	"MAX_TEMPO":              240.0,
	"MIN_EQ_FREQ":            20.0,
	"MAX_EQ_FREQ":            20000.0,
}

// ParamSpec is one editable parameter of a model, with the scale the device's
// own catalog publishes: min, max and skew describe every knob, and
//
//	wire = ((real - min) / (max - min)) ** skew
//
// is the single conversion law (real on the screen's line, wire the 0..1 the
// device stores). See pyquadcortex's catalog docs for the hardware evidence.
type ParamSpec struct {
	Name      string
	Type      string
	Units     string
	Min       float64
	Max       float64
	Default   float64
	Skew      float64
	Steps     int
	StepNames []string
	MinLabel  string
	MaxLabel  string
	// unmeasured marks a parameter whose bounds the catalog names but nobody
	// has measured; Normalize/Denormalize refuse it instead of guessing.
	unmeasured bool
}

// Normalize converts a value in the parameter's own units to the wire's 0..1,
// applying the catalog's skew. Values outside the range are refused.
func (p ParamSpec) Normalize(real float64) (float64, error) {
	if p.unmeasured {
		return 0, fmt.Errorf("%s: bounds are unmeasured; pass an encoded 0..1 value", p.Name)
	}
	span := p.Max - p.Min
	if span <= 0 {
		return 0, fmt.Errorf("%s: degenerate range %g..%g", p.Name, p.Min, p.Max)
	}
	fraction := (real - p.Min) / span
	fraction = math.Min(1, math.Max(0, fraction))
	return math.Pow(fraction, p.Skew), nil
}

// Denormalize converts a wire 0..1 value back to the parameter's own units.
func (p ParamSpec) Denormalize(wire float64) (float64, error) {
	if p.unmeasured {
		return 0, fmt.Errorf("%s: bounds are unmeasured", p.Name)
	}
	if p.Skew <= 0 {
		return 0, fmt.Errorf("%s: invalid skew %g", p.Name, p.Skew)
	}
	span := p.Max - p.Min
	w := math.Min(1, math.Max(0, wire))
	return p.Min + span*math.Pow(w, 1/p.Skew), nil
}

// OptionToValue maps an option index to the wire value that selects it. List
// parameters (switch, comboBox, rotarySwitch) are stored as an evenly spaced
// 0..1 value: option i sits at i/(count-1).
func (p ParamSpec) OptionToValue(option int) (float64, error) {
	if p.Steps <= 0 {
		return 0, fmt.Errorf("%s is not a list parameter", p.Name)
	}
	if option < 0 || option >= p.Steps {
		return 0, fmt.Errorf("%s has %d options (0..%d), got %d", p.Name, p.Steps, p.Steps-1, option)
	}
	if p.Steps == 1 {
		return 0, nil
	}
	return float64(option) / float64(p.Steps-1), nil
}

// OptionName looks up a list parameter's option by name (case-insensitive).
func (p ParamSpec) OptionName(name string) (int, error) {
	if p.Steps <= 0 {
		return 0, fmt.Errorf("%s is not a list parameter", p.Name)
	}
	for i, n := range p.StepNames {
		if strings.EqualFold(n, name) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("%s has no option %q (options: %s)", p.Name, name, strings.Join(p.StepNames, ", "))
}

// ModelSpec is one block type: an amp, a pedal, a cab or a capture. It is
// named ModelSpec because the protobuf package already owns the name Model.
type ModelSpec struct {
	ID       int
	Name     string
	Category string
	BasedOn  string
	Params   []ParamSpec
}

// Param returns the parameter with the given (case-insensitive) name.
func (m ModelSpec) Param(name string) (ParamSpec, int, bool) {
	for i, p := range m.Params {
		if strings.EqualFold(p.Name, name) {
			return p, i, true
		}
	}
	return ParamSpec{}, 0, false
}

// Catalog is the parsed ModelRepo, keyed by wire hash.
type Catalog struct {
	byID   map[int]*ModelSpec
	byName map[string][]*ModelSpec
}

// Model returns the model with the given wire hash.
func (c *Catalog) Model(id int) (*ModelSpec, bool) {
	m, ok := c.byID[id]
	return m, ok
}

// Find returns the first model whose name equals query (case-insensitive), or
// whose "based on" text contains it. Among substring matches the lowest id
// wins, so the base variant of a channel pair resolves before its "- Vibrato"
// or "- Lead" siblings.
func (c *Catalog) Find(query string) (*ModelSpec, bool) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, false
	}
	for _, m := range c.byName {
		for _, m := range m {
			if strings.EqualFold(m.Name, q) {
				return m, true
			}
		}
	}
	var best *ModelSpec
	for _, list := range c.byName {
		for _, m := range list {
			if m.BasedOn != "" && strings.Contains(strings.ToLower(m.BasedOn), strings.ToLower(q)) {
				if best == nil || m.ID < best.ID {
					best = m
				}
			}
		}
	}
	return best, best != nil
}

var (
	catalogOnce sync.Once
	catalogVal  *Catalog
	catalogErr  error
)

// defaultCatalog parses the embedded ModelRepo exactly once.
func defaultCatalog() (*Catalog, error) {
	catalogOnce.Do(func() {
		catalogVal, catalogErr = parseModelRepo(modelRepoXML)
	})
	return catalogVal, catalogErr
}

func parseModelRepo(data []byte) (*Catalog, error) {
	var root xmlModels
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("qc: parse ModelRepo: %w", err)
	}
	c := &Catalog{byID: map[int]*ModelSpec{}, byName: map[string][]*ModelSpec{}}
	for _, cat := range root.Categories {
		for _, xm := range cat.Models {
			m := &ModelSpec{
				ID:       xm.ID,
				Name:     strings.TrimSpace(xm.Name),
				Category: cat.Name,
				BasedOn:  cleanBasedOn(xm.TM),
			}
			for _, xp := range xm.Params {
				// "empty" parameters are wire padding, not knobs; they carry no
				// bounds and must not become addressable parameters.
				if strings.TrimSpace(xp.Type) == "empty" || strings.TrimSpace(xp.Name) == "" {
					continue
				}
				spec, err := parseParamSpec(xp)
				if err != nil {
					return nil, fmt.Errorf("qc: model %q parameter %q: %w", xm.Name, xp.Name, err)
				}
				m.Params = append(m.Params, spec)
			}
			c.byID[m.ID] = m
			c.byName[m.Name] = append(c.byName[m.Name], m)
		}
	}
	return c, nil
}

func cleanBasedOn(tm string) string {
	s := strings.TrimSpace(tm)
	s = strings.TrimPrefix(s, "Based on ")
	s = strings.TrimPrefix(s, "Based on")
	s = strings.TrimSpace(s)
	s = strings.NewReplacer("®", "", "™", "", "�", "").Replace(s)
	return strings.TrimSpace(s)
}

func parseParamSpec(p xmlParam) (ParamSpec, error) {
	spec := ParamSpec{
		Name:      strings.TrimSpace(p.Name),
		Type:      strings.TrimSpace(p.Type),
		Units:     strings.TrimSpace(p.Units),
		MinLabel:  strings.TrimSpace(p.MinString),
		MaxLabel:  strings.TrimSpace(p.MaxString),
		StepNames: splitStepNames(p.StepNames),
	}
	if v, err := parseFloat(p.DefaultValue, 0); err == nil {
		spec.Default = v
	}
	if v, err := parseFloat(p.Steps, 0); err == nil {
		spec.Steps = int(v)
	}
	skew, err := parseSkew(p.Skew)
	if err != nil {
		return ParamSpec{}, err
	}
	spec.Skew = skew

	min, minMeasured, err := resolveBound(p.Min)
	if err != nil {
		return ParamSpec{}, err
	}
	max, maxMeasured, err := resolveBound(p.Max)
	if err != nil {
		return ParamSpec{}, err
	}
	spec.unmeasured = !minMeasured || !maxMeasured
	spec.Min = min
	spec.Max = max
	return spec, nil
}

// resolveBound turns a numeric or symbolic bound into a number. A symbolic
// name the constants do not know is UNMEASURED: the value stays zero and
// measured is false so conversions refuse instead of guessing.
func resolveBound(raw string) (value float64, measured bool, err error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, false, fmt.Errorf("a bound is empty")
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v, true, nil
	}
	if v, ok := firmwareConstants[s]; ok {
		return v, true, nil
	}
	if strings.HasPrefix(s, "MIN_") || strings.HasPrefix(s, "MAX_") {
		// A named bound nobody has measured: mark it unmeasured rather than
		// inventing a range.
		return 0, false, nil
	}
	return 0, false, fmt.Errorf("unrecognised bound %q", s)
}

func parseSkew(raw string) (float64, error) {
	text := strings.TrimSpace(raw)
	if text == "" || text == "LIN_SKEW" {
		return 1.0, nil
	}
	if text == "LOG_SKEW" {
		// Not a logarithmic sweep: the same power law, at skew 0.3 (confirmed
		// on hardware).
		return 0.3, nil
	}
	v, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("unknown taper %q", raw)
	}
	if !(v > 0 && !math.IsInf(v, 0) && !math.IsNaN(v)) {
		return 0, fmt.Errorf("invalid taper %q", raw)
	}
	return v, nil
}

func parseFloat(s string, fallback float64) (float64, error) {
	if strings.TrimSpace(s) == "" {
		return fallback, fmt.Errorf("empty number")
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return fallback, err
	}
	return v, nil
}

func splitStepNames(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

type xmlModels struct {
	Categories []xmlCategory `xml:"Category"`
}

type xmlCategory struct {
	ID     int        `xml:"id,attr"`
	Name   string     `xml:"name,attr"`
	Models []xmlModel `xml:"Model"`
}

type xmlModel struct {
	ID     int        `xml:"id,attr"`
	Name   string     `xml:"name,attr"`
	TM     string     `xml:"tm,attr"`
	Params []xmlParam `xml:"Parameter"`
}

type xmlParam struct {
	Name         string `xml:"name,attr"`
	Type         string `xml:"type,attr"`
	Min          string `xml:"min,attr"`
	Max          string `xml:"max,attr"`
	DefaultValue string `xml:"defaultValue,attr"`
	Skew         string `xml:"skew,attr"`
	Steps        string `xml:"steps,attr"`
	StepNames    string `xml:"stepNames,attr"`
	Units        string `xml:"units,attr"`
	MinString    string `xml:"min_string,attr"`
	MaxString    string `xml:"max_string,attr"`
}
