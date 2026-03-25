package issues

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// parseFrontmatter extracts the raw YAML string from a file's content.
// Returns the yaml string (without --- delimiters) and the remainder of the file.
func parseFrontmatter(content string) (yamlStr, rest string, err error) {
	if !strings.HasPrefix(content, "---\n") {
		return "", content, nil
	}
	after := content[4:] // skip leading "---\n"
	idx := strings.Index(after, "\n---\n")
	if idx != -1 {
		return after[:idx], after[idx+5:], nil
	}
	// Check for closing --- at end of file (no trailing newline)
	if strings.HasSuffix(after, "\n---") {
		return after[:len(after)-4], "", nil
	}
	return "", content, fmt.Errorf("unclosed frontmatter block")
}

// unmarshalFrontmatter parses a YAML string into a map.
func unmarshalFrontmatter(yamlStr string) (map[string]interface{}, error) {
	var fields map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		fields = make(map[string]interface{})
	}
	return fields, nil
}

// marshalFrontmatter serializes a fields map to a YAML string (without --- delimiters).
func marshalFrontmatter(fields map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(fields); err != nil {
		return "", err
	}
	enc.Close()
	return buf.String(), nil
}
