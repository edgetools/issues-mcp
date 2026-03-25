package issues

import (
	"strings"
)

// SearchMatch is a single line match within an issue file.
type SearchMatch struct {
	Line    int    `json:"line"`
	Context string `json:"context"`
}

// SearchResult is a matched issue with its match locations.
type SearchResult struct {
	ID      string        `json:"id"`
	Title   string        `json:"title"`
	Status  string        `json:"status"`
	Area    string        `json:"area"`
	Path    string        `json:"path"`
	Matches []SearchMatch `json:"matches"`
}

// Search performs a case-insensitive keyword search across all issue files.
// Multiple keywords are each matched independently; a line matching any keyword is included.
// statusFilter restricts the search to a single status directory when non-empty.
func (st *Store) Search(query, statusFilter string) ([]SearchResult, error) {
	files, err := st.ListAll(statusFilter, "")
	if err != nil {
		return nil, err
	}

	keywords := strings.Fields(strings.ToLower(query))
	if len(keywords) == 0 {
		return nil, nil
	}

	var results []SearchResult
	for _, f := range files {
		matches := searchInContent(f.RawContent, keywords)
		if len(matches) > 0 {
			results = append(results, SearchResult{
				ID:      f.ID,
				Title:   StringField(f.Fields, "title"),
				Status:  f.DirStatus,
				Area:    StringField(f.Fields, "area"),
				Path:    f.RelPath,
				Matches: matches,
			})
		}
	}
	return results, nil
}

// searchInContent finds all lines matching any keyword in the given content.
// Returns one SearchMatch per matching line (no duplicates).
func searchInContent(content string, keywords []string) []SearchMatch {
	lines := strings.Split(content, "\n")
	seen := make(map[int]bool)
	var matches []SearchMatch

	for lineIdx, line := range lines {
		lower := strings.ToLower(line)
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				lineNum := lineIdx + 1
				if !seen[lineNum] {
					seen[lineNum] = true
					matches = append(matches, SearchMatch{
						Line:    lineNum,
						Context: strings.TrimSpace(line),
					})
				}
				break
			}
		}
	}
	return matches
}
