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

func TestDecodeJSON_UnknownField(t *testing.T) {
	reader := strings.NewReader(`{"field": "value", "unknown": "extra"}`)
	var target dummyStruct
	if err := DecodeJSON(reader, &target); err == nil {
		t.Error("DecodeJSON() expected error for unknown field")
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
