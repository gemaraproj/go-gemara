// SPDX-License-Identifier: Apache-2.0

package codec

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/goccy/go-yaml"
)

// DecodeYAML decodes YAML from a reader into the target.
func DecodeYAML(reader io.Reader, target interface{}) error {
	if err := yaml.NewDecoder(reader).Decode(target); err != nil {
		return fmt.Errorf("error decoding YAML: %w", err)
	}
	return nil
}

// DecodeJSON decodes JSON from a reader into the target.
// Unknown fields in the input are ignored, so artifacts written against a
// newer Gemara schema still decode into the types this version supports.
func DecodeJSON(reader io.Reader, target interface{}) error {
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("error decoding JSON: %w", err)
	}
	return nil
}

// MarshalYAML marshals an object to YAML bytes.
func MarshalYAML(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

// UnmarshalYAML unmarshals YAML bytes into the provided target.
// Unknown fields in the input are ignored.
func UnmarshalYAML(data []byte, target interface{}) error {
	return yaml.Unmarshal(data, target)
}
