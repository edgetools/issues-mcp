package schema_test

import (
	"testing"

	"github.com/edgetools/issues-mcp/schema"
)

func makeRecord(id, area, dirStatus string, extra map[string]any) schema.IssueRecord {
	fields := map[string]any{
		"id":         id,
		"area":       area,
		"status":     dirStatus,
		"depends_on": []interface{}{},
	}
	for k, v := range extra {
		fields[k] = v
	}
	return schema.IssueRecord{ID: id, Fields: fields, DirStatus: dirStatus}
}

func minimalSchema() *schema.Schema {
	return &schema.Schema{Statuses: []string{"todo", "done"}}
}

func TestValidateAll_CircularDependency(t *testing.T) {
	s := minimalSchema()

	a := makeRecord("TODO-001", "Todo", "todo", map[string]any{
		"depends_on": []interface{}{"TODO-002"},
	})
	b := makeRecord("TODO-002", "Todo", "todo", map[string]any{
		"depends_on": []interface{}{"TODO-001"},
	})

	errs := schema.ValidateAll(s, []schema.IssueRecord{a, b})
	found := false
	for _, e := range errs {
		if e.Field == "depends_on" && e.Error == "circular dependency detected" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected circular dependency error, got errors: %+v", errs)
	}
}

func TestValidateAll_ThreeWayCircularDependency(t *testing.T) {
	s := minimalSchema()

	a := makeRecord("TODO-001", "Todo", "todo", map[string]any{"depends_on": []interface{}{"TODO-002"}})
	b := makeRecord("TODO-002", "Todo", "todo", map[string]any{"depends_on": []interface{}{"TODO-003"}})
	c := makeRecord("TODO-003", "Todo", "todo", map[string]any{"depends_on": []interface{}{"TODO-001"}})

	errs := schema.ValidateAll(s, []schema.IssueRecord{a, b, c})
	found := false
	for _, e := range errs {
		if e.Field == "depends_on" && e.Error == "circular dependency detected" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected circular dependency error for 3-way cycle, got errors: %+v", errs)
	}
}

func TestValidateAll_LinearChainIsNotCircular(t *testing.T) {
	s := minimalSchema()

	a := makeRecord("TODO-001", "Todo", "todo", map[string]any{"depends_on": []interface{}{"TODO-002"}})
	b := makeRecord("TODO-002", "Todo", "todo", map[string]any{"depends_on": []interface{}{}})

	errs := schema.ValidateAll(s, []schema.IssueRecord{a, b})
	for _, e := range errs {
		if e.Field == "depends_on" && e.Error == "circular dependency detected" {
			t.Errorf("unexpected circular dependency error for linear chain: %+v", e)
		}
	}
}

func TestValidateRecord_DependsOnUnknownID(t *testing.T) {
	s := minimalSchema()

	rec := makeRecord("TODO-001", "Todo", "todo", map[string]any{
		"depends_on": []interface{}{"NONEXISTENT-001"},
	})
	allIDs := map[string]bool{"TODO-001": true}

	errs := schema.ValidateRecord(s, rec, allIDs)
	found := false
	for _, e := range errs {
		if e.Field == "depends_on" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected depends_on error for unknown ID, got errors: %+v", errs)
	}
}

func TestValidateRecord_DependsOnKnownID(t *testing.T) {
	s := minimalSchema()

	a := makeRecord("TODO-001", "Todo", "todo", map[string]any{"depends_on": []interface{}{"TODO-002"}})
	allIDs := map[string]bool{"TODO-001": true, "TODO-002": true}

	errs := schema.ValidateRecord(s, a, allIDs)
	for _, e := range errs {
		if e.Field == "depends_on" {
			t.Errorf("unexpected depends_on error for known ID: %+v", e)
		}
	}
}
