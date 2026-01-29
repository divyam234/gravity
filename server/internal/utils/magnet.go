package utils

import (
	"strings"
)

// ExtractHashFromMagnet extracts btih hash from magnet URI
func ExtractHashFromMagnet(magnet string) string {
	lower := strings.ToLower(magnet)
	if idx := strings.Index(lower, "btih:"); idx >= 0 {
		hash := magnet[idx+5:]
		if ampIdx := strings.Index(hash, "&"); ampIdx >= 0 {
			hash = hash[:ampIdx]
		}
		return strings.ToLower(hash)
	}
	return ""
}
