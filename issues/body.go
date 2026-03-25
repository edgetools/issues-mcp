package issues

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const workLogSeparator = "<!-- work-log -->"
const workLogHeading = "### Work Log"

// WorkLogEntry is a single parsed work log entry.
type WorkLogEntry struct {
	Timestamp string `json:"timestamp"`
	Author    string `json:"author"`
	Content   string `json:"content"`
}

// IssueFile is a fully parsed issue file held in memory.
type IssueFile struct {
	Fields     map[string]interface{}
	Body       string
	WorkLog    []WorkLogEntry
	AbsPath    string // absolute path on disk
	RelPath    string // path relative to issues-dir (forward slashes)
	DirStatus  string // status derived from directory name
	ID         string
	RawContent string // original file bytes, used for search
}

// splitSections splits raw file content into frontmatter yaml, body, and work log text.
func splitSections(content string) (fmYAML, body, workLogRaw string, err error) {
	fmYAML, rest, err := parseFrontmatter(content)
	if err != nil {
		return "", "", "", err
	}
	idx := strings.Index(rest, workLogSeparator)
	if idx == -1 {
		return fmYAML, strings.TrimRight(rest, "\n"), "", nil
	}
	body = strings.TrimRight(rest[:idx], "\n")
	workLogRaw = strings.TrimLeft(rest[idx+len(workLogSeparator):], "\n")
	return fmYAML, body, workLogRaw, nil
}

// ParseIssueFile parses raw file content into an IssueFile.
func ParseIssueFile(content string) (IssueFile, error) {
	fmYAML, body, workLogRaw, err := splitSections(content)
	if err != nil {
		return IssueFile{}, err
	}
	fields, err := unmarshalFrontmatter(fmYAML)
	if err != nil {
		return IssueFile{}, fmt.Errorf("parsing frontmatter: %w", err)
	}
	return IssueFile{
		Fields:     fields,
		Body:       strings.TrimSpace(body),
		WorkLog:    parseWorkLog(workLogRaw),
		RawContent: content,
	}, nil
}

// BuildContent serializes an IssueFile back to raw file content.
func (f *IssueFile) BuildContent() (string, error) {
	fmYAML, err := marshalFrontmatter(f.Fields)
	if err != nil {
		return "", fmt.Errorf("marshaling frontmatter: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmYAML)
	sb.WriteString("---\n")

	if f.Body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(f.Body, "\n"))
		sb.WriteString("\n")
	}

	if len(f.WorkLog) > 0 {
		sb.WriteString("\n")
		sb.WriteString(workLogSeparator)
		sb.WriteString("\n\n")
		sb.WriteString(workLogHeading)
		sb.WriteString("\n")
		for _, entry := range f.WorkLog {
			sb.WriteString("\n")
			sb.WriteString(fmt.Sprintf("**%s | %s**\n", entry.Timestamp, entry.Author))
			content := strings.TrimRight(entry.Content, "\n")
			sb.WriteString(content)
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

// AppendWorkLogEntry adds a new entry to the work log, returning the generated timestamp.
func (f *IssueFile) AppendWorkLogEntry(author, content string) string {
	ts := time.Now().UTC().Format(time.RFC3339)
	f.WorkLog = append(f.WorkLog, WorkLogEntry{
		Timestamp: ts,
		Author:    author,
		Content:   strings.TrimSpace(content),
	})
	return ts
}

var workLogEntryHeader = regexp.MustCompile(`^\*\*([^|*]+?)\s*\|\s*([^*]+?)\*\*\s*$`)

// parseWorkLog parses the text after the work log separator into structured entries.
func parseWorkLog(raw string) []WorkLogEntry {
	if raw == "" {
		return nil
	}
	lines := strings.Split(raw, "\n")
	var entries []WorkLogEntry
	var current *WorkLogEntry
	var contentLines []string

	flush := func() {
		if current != nil {
			current.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))
			entries = append(entries, *current)
			current = nil
			contentLines = nil
		}
	}

	for _, line := range lines {
		if m := workLogEntryHeader.FindStringSubmatch(line); m != nil {
			flush()
			current = &WorkLogEntry{
				Timestamp: strings.TrimSpace(m[1]),
				Author:    strings.TrimSpace(m[2]),
			}
		} else if current != nil {
			if line == workLogHeading {
				continue // skip the heading line
			}
			contentLines = append(contentLines, line)
		}
	}
	flush()

	return entries
}
