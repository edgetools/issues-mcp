package issues

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/edgetools/issues-mcp/schema"
)

// Store manages reading, writing, and moving issue files on disk.
type Store struct {
	IssuesDir string
	Schema    *schema.Schema
}

// NewStore creates a Store and ensures all status subdirectories exist.
func NewStore(issuesDir string, s *schema.Schema) (*Store, error) {
	st := &Store{IssuesDir: issuesDir, Schema: s}
	for _, status := range s.Statuses {
		dir := filepath.Join(issuesDir, status)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("creating directory %q: %w", dir, err)
		}
	}
	return st, nil
}

// FindByID locates an issue by ID, scanning all status directories.
func (st *Store) FindByID(id string) (*IssueFile, error) {
	filename := id + ".md"
	for _, status := range st.Schema.Statuses {
		dir := filepath.Join(st.IssuesDir, status)
		entries, err := listDir(dir)
		if err != nil {
			continue
		}
		for _, name := range entries {
			if name == filename {
				return st.ReadFile(filepath.Join(dir, name), status)
			}
		}
	}
	return nil, fmt.Errorf("issue %q not found", id)
}

// ListAll returns all issues, optionally filtered by status and/or area prefix.
// If statusFilter is empty, all statuses are included.
// If areaFilter is non-empty, only issues whose area starts with areaFilter are returned.
func (st *Store) ListAll(statusFilter, areaFilter string) ([]*IssueFile, error) {
	statuses := st.Schema.Statuses
	if statusFilter != "" {
		// Validate the filter value
		found := false
		for _, s := range st.Schema.Statuses {
			if s == statusFilter {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown status %q", statusFilter)
		}
		statuses = []string{statusFilter}
	}

	var results []*IssueFile
	for _, status := range statuses {
		dir := filepath.Join(st.IssuesDir, status)
		entries, err := listDir(dir)
		if err != nil {
			continue
		}
		for _, name := range entries {
			if !strings.HasSuffix(name, ".md") {
				continue
			}
			f, err := st.ReadFile(filepath.Join(dir, name), status)
			if err != nil {
				continue
			}
			if areaFilter != "" {
				area, _ := f.Fields["area"].(string)
				if !strings.HasPrefix(area, areaFilter) {
					continue
				}
			}
			results = append(results, f)
		}
	}
	return results, nil
}

// ReadFile reads and parses one issue file from disk.
func (st *Store) ReadFile(absPath, dirStatus string) (*IssueFile, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", absPath, err)
	}
	f, err := ParseIssueFile(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing %q: %w", absPath, err)
	}
	f.AbsPath = absPath
	f.DirStatus = dirStatus
	relPath, _ := filepath.Rel(st.IssuesDir, absPath)
	f.RelPath = filepath.ToSlash(relPath)
	if id, ok := f.Fields["id"].(string); ok {
		f.ID = id
	} else {
		base := filepath.Base(absPath)
		f.ID = strings.TrimSuffix(base, ".md")
	}
	return &f, nil
}

// Write serializes and writes an issue file to its current AbsPath.
func (st *Store) Write(f *IssueFile) error {
	content, err := f.BuildContent()
	if err != nil {
		return err
	}
	return os.WriteFile(f.AbsPath, []byte(content), 0644)
}

// CheckAndMove validates the transition and gate conditions, then moves the issue to newStatus.
// The caller is responsible for having already applied any field updates to f.Fields before calling.
func (st *Store) CheckAndMove(f *IssueFile, newStatus string) error {
	// Validate target status exists
	validStatus := false
	for _, s := range st.Schema.Statuses {
		if s == newStatus {
			validStatus = true
			break
		}
	}
	if !validStatus {
		return fmt.Errorf("unknown status %q", newStatus)
	}

	// Check transition is allowed
	current := f.DirStatus
	allowed := st.Schema.Transitions[current]
	permitted := false
	for _, a := range allowed {
		if a == newStatus {
			permitted = true
			break
		}
	}
	if !permitted {
		return fmt.Errorf("cannot move %s from '%s' to '%s': transition not allowed (%s allows: [%s])",
			f.ID, current, newStatus, current, strings.Join(allowed, ", "))
	}

	// Check gate conditions
	if gate, ok := st.Schema.Gates[newStatus]; ok {
		for field, required := range gate.Requires {
			actual := f.Fields[field]
			if !schema.ValuesEqual(actual, required) {
				return fmt.Errorf("cannot move %s to '%s': gate requires %s=%v but current value is %v",
					f.ID, newStatus, field, required, actual)
			}
		}
	}

	return st.move(f, newStatus)
}

// move physically moves the issue file to the new status directory and updates in-memory state.
func (st *Store) move(f *IssueFile, newStatus string) error {
	newDir := filepath.Join(st.IssuesDir, newStatus)
	newPath := filepath.Join(newDir, filepath.Base(f.AbsPath))

	// Update frontmatter status field before writing
	f.Fields["status"] = newStatus

	content, err := f.BuildContent()
	if err != nil {
		return err
	}
	if err := os.WriteFile(newPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing to %q: %w", newPath, err)
	}
	if err := os.Remove(f.AbsPath); err != nil {
		// Roll back: remove the new file so the old one is still canonical
		os.Remove(newPath)
		// Revert in-memory status
		f.Fields["status"] = f.DirStatus
		return fmt.Errorf("removing old file %q: %w", f.AbsPath, err)
	}

	// Update in-memory state to reflect new location
	f.AbsPath = newPath
	f.DirStatus = newStatus
	relPath, _ := filepath.Rel(st.IssuesDir, newPath)
	f.RelPath = filepath.ToSlash(relPath)
	return nil
}

// AllRecords returns all issues as schema.IssueRecord values for bulk validation.
func (st *Store) AllRecords() ([]schema.IssueRecord, error) {
	files, err := st.ListAll("", "")
	if err != nil {
		return nil, err
	}
	records := make([]schema.IssueRecord, 0, len(files))
	for _, f := range files {
		records = append(records, schema.IssueRecord{
			ID:        f.ID,
			Fields:    f.Fields,
			DirStatus: f.DirStatus,
		})
	}
	return records, nil
}

// listDir returns the base filenames in a directory, ignoring missing directories.
func listDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// StringField safely extracts a string value from a fields map.
func StringField(fields map[string]interface{}, key string) string {
	if v, ok := fields[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
