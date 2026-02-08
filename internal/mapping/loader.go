package mapping

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadFile loads and parses a YAML mapping file from the given path.
func LoadFile(path string) (*MappingFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping file %s: %w", path, err)
	}

	return Parse(data)
}

// Parse parses YAML data into a MappingFile.
func Parse(data []byte) (*MappingFile, error) {
	var mf MappingFile

	err := yaml.Unmarshal(data, &mf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse mapping YAML: %w", err)
	}

	// Apply defaults and normalize
	applyDefaults(&mf)

	return &mf, nil
}

// applyDefaults fills in default values for optional fields.
func applyDefaults(mf *MappingFile) {
	if mf.Version == "" {
		mf.Version = "1"
	}

	for i := range mf.Transforms {
		t := &mf.Transforms[i]
		if t.Func == "" {
			t.Func = t.Name
		}
	}
}

// Marshal serializes a MappingFile to YAML.
func Marshal(mf *MappingFile) ([]byte, error) {
	return yaml.Marshal(mf)
}

// WriteFile writes a MappingFile to the given path.
func WriteFile(mf *MappingFile, path string) error {
	data, err := Marshal(mf)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write mapping file %s: %w", path, err)
	}

	return nil
}

// NormalizeTypeMapping expands 121 shorthand into Fields entries
// and ensures all mappings are in canonical form.
func NormalizeTypeMapping(tm *TypeMapping) {
	// Convert 121 shorthand to explicit field mappings
	// These are prepended to Fields so they have higher effective priority
	// (when resolving, 121 entries will be checked first)
	if len(tm.OneToOne) > 0 {
		expanded := make([]FieldMapping, 0, len(tm.OneToOne))

		for source, target := range tm.OneToOne {
			expanded = append(expanded, FieldMapping{
				Source: FieldRefArray{{Path: source, Hint: HintNone}},
				Target: FieldRefArray{{Path: target, Hint: HintNone}},
			})
		}

		// 121 has highest priority, so it goes first
		tm.Fields = append(expanded, tm.Fields...)
	}
}

// NormalizeMappingFile normalizes all type mappings in a file.
func NormalizeMappingFile(mf *MappingFile) {
	for i := range mf.TypeMappings {
		NormalizeTypeMapping(&mf.TypeMappings[i])
	}
}
