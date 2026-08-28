package responses

import (
	"strings"
)

// isLegacyCursor reports whether the User-Agent indicates an older Cursor
// version that may not fully support Responses API reasoning output items.
// For these clients, we skip emitting reasoning items even when requested
// to prevent silent rendering failures.
func isLegacyCursor(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	if !strings.Contains(ua, "cursor") {
		return false
	}
	// "Cursor/1.0" is a generic early identifier; newer versions use more
	// specific version strings like "Cursor/0.42.1" or higher.
	if strings.Contains(ua, "cursor/1.0") {
		return true
	}
	// Additional legacy patterns can be added here as needed.
	return false
}
