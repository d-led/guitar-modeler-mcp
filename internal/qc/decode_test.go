package qc

import (
	"testing"

	"google.golang.org/protobuf/proto"
)

// samplePreset builds a minimal but realistic BinaryPreset: one row with one
// amp model carrying a float parameter, plus two scene labels.
func samplePreset() *BinaryPreset {
	return &BinaryPreset{
		Name:   "JCM800 Crunch",
		Tempo:  120,
		Volume: -6,
		Pan:    0,
		Chains: []*Chain{
			{
				XInPortid:  &Chain_InPortid{InPortid: 1},
				XOutPortid: &Chain_OutPortid{OutPortid: 0},
				XRow:       &Chain_Row{Row: 0},
				Models: []*Model{
					{
						XHash:   &Model_Hash{Hash: 21005},
						XColumn: &Model_Column{Column: 0},
						Params: []*Param{
							{
								XIndex:      &Param_Index{Index: 0},
								ParamValues: []*ParamValue{{Value: &ParamValue_FloatValue{FloatValue: 0.5}}},
							},
						},
					},
				},
			},
		},
		SceneLabels: []string{"Scene A", "Scene B"},
	}
}

func TestEncodeDecodePresetRoundTrip(t *testing.T) {
	want := samplePreset()
	data, err := EncodePreset("QA00XXXXX", want)
	if err != nil {
		t.Fatalf("EncodePreset: %v", err)
	}

	got, err := DecodePreset("QA00XXXXX", data)
	if err != nil {
		t.Fatalf("DecodePreset: %v", err)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("round trip changed the preset:\n got %v\nwant %v", got, want)
	}
}

func TestDecodePresetRejectsWrongSerial(t *testing.T) {
	data, err := EncodePreset("QA00XXXXX", samplePreset())
	if err != nil {
		t.Fatalf("EncodePreset: %v", err)
	}
	// A different (or empty) serial derives a different stream, so the result
	// is not a valid BinaryPreset and must fail to decode rather than succeed
	// with garbage.
	if _, err := DecodePreset("QB99YYYYY", data); err == nil {
		t.Fatal("DecodePreset with the wrong serial succeeded, want an error")
	}
}

func TestDecodePresetRejectsPlaintext(t *testing.T) {
	// Raw (unencrypted) protobuf bytes are not a valid encrypted file: CTR
	// turns them into noise that cannot unmarshal.
	plain, err := proto.Marshal(samplePreset())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if _, err := DecodePreset("QA00XXXXX", plain); err == nil {
		t.Fatal("DecodePreset on plaintext succeeded, want an error")
	}
}
