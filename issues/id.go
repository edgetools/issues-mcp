package issues

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// AreaToPrefix converts an area string to an ID prefix.
// "Combat/Aggro" → "COMBAT-AGGRO"
// "Entity" → "ENTITY"
func AreaToPrefix(area string) string {
	parts := strings.Split(area, "/")
	for i, p := range parts {
		parts[i] = strings.ToUpper(p)
	}
	return strings.Join(parts, "-")
}

// NextID generates the next available ID for a given area by scanning all status directories.
func NextID(issuesDir string, statuses []string, area string) (string, error) {
	prefix := AreaToPrefix(area)
	pattern := regexp.MustCompile(fmt.Sprintf(`^%s-(\d+)\.md$`, regexp.QuoteMeta(prefix)))

	max := 0
	for _, status := range statuses {
		dir := filepath.Join(issuesDir, status)
		entries, err := listDir(dir)
		if err != nil {
			continue // directory may not exist
		}
		for _, name := range entries {
			m := pattern.FindStringSubmatch(name)
			if m == nil {
				continue
			}
			n, err := strconv.Atoi(m[1])
			if err == nil && n > max {
				max = n
			}
		}
	}
	return fmt.Sprintf("%s-%03d", prefix, max+1), nil
}
