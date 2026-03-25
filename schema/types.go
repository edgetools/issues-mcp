package schema

import "encoding/json"

// FieldType enumerates the supported frontmatter field types.
type FieldType string

const (
	FieldTypeString FieldType = "string"
	FieldTypeBool   FieldType = "bool"
	FieldTypeEnum   FieldType = "enum"
	FieldTypeList   FieldType = "list"
	FieldTypeInt    FieldType = "int"
)

// FieldDef describes a single frontmatter field from schema.json.
type FieldDef struct {
	Type      FieldType       `json:"type"`
	Required  bool            `json:"required"`
	Generated bool            `json:"generated"`
	Values    []string        `json:"values"`    // enum valid values
	ItemType  string          `json:"item_type"` // list element type
	Default   json.RawMessage `json:"default"`   // JSON-encoded default value
}

// GateDef is a set of field conditions that must be met to enter a status.
type GateDef struct {
	Requires map[string]interface{} `json:"requires"`
}

// BodyLock defines which statuses lock the issue body against edits.
type BodyLock struct {
	LockedIn []string `json:"locked_in"`
}

// Locks groups all lock definitions.
type Locks struct {
	Body *BodyLock `json:"body"`
}

// Schema is the parsed schema.json.
type Schema struct {
	Statuses    []string            `json:"statuses"`
	Fields      map[string]FieldDef `json:"fields"`
	Transitions map[string][]string `json:"transitions"`
	Gates       map[string]GateDef  `json:"gates"`
	Locks       Locks               `json:"locks"`
}
