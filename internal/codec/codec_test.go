// SPDX-License-Identifier: Apache-2.0

package codec

import (
	"strings"
	"testing"
)

type dummyStruct struct {
	Field string `yaml:"field" json:"field"`
}

func TestDecodeYAML(t *testing.T) {
	reader := strings.NewReader("field: value\n")
	var target dummyStruct
	if err := DecodeYAML(reader, &target); err != nil {
		t.Fatalf("DecodeYAML() error = %v", err)
	}
	if target.Field != "value" {
		t.Errorf("DecodeYAML() got = %v, want %v", target.Field, "value")
	}
}

func TestDecodeYAML_Invalid(t *testing.T) {
	reader := strings.NewReader(":\n  bad yaml {{{\n")
	var target dummyStruct
	if err := DecodeYAML(reader, &target); err == nil {
		t.Error("DecodeYAML() expected error for invalid YAML")
	}
}

func TestDecodeJSON(t *testing.T) {
	reader := strings.NewReader(`{"field": "value"}`)
	var target dummyStruct
	if err := DecodeJSON(reader, &target); err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}
	if target.Field != "value" {
		t.Errorf("DecodeJSON() got = %v, want %v", target.Field, "value")
	}
}

// Unknown fields must be ignored so that artifacts written against a newer
// schema still decode into the types this version of the library ships.
func TestDecodeJSON_UnknownFieldIgnored(t *testing.T) {
	reader := strings.NewReader(`{"field": "value", "future-key": "extra"}`)
	var target dummyStruct
	if err := DecodeJSON(reader, &target); err != nil {
		t.Fatalf("DecodeJSON() error = %v, want nil for unknown field", err)
	}
	if target.Field != "value" {
		t.Errorf("DecodeJSON() got = %v, want %v", target.Field, "value")
	}
}

func TestDecodeYAML_UnknownFieldIgnored(t *testing.T) {
	reader := strings.NewReader("field: value\nfuture-key: extra\n")
	var target dummyStruct
	if err := DecodeYAML(reader, &target); err != nil {
		t.Fatalf("DecodeYAML() error = %v, want nil for unknown field", err)
	}
	if target.Field != "value" {
		t.Errorf("DecodeYAML() got = %v, want %v", target.Field, "value")
	}
}

func TestMarshalUnmarshalYAML(t *testing.T) {
	obj := dummyStruct{Field: "value"}
	data, err := MarshalYAML(obj)
	if err != nil {
		t.Fatalf("MarshalYAML() error = %v", err)
	}
	var target dummyStruct
	if err := UnmarshalYAML(data, &target); err != nil {
		t.Fatalf("UnmarshalYAML() error = %v", err)
	}
	if target.Field != "value" {
		t.Errorf("UnmarshalYAML() got = %v, want %v", target.Field, "value")
	}
}
