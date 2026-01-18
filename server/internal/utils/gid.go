package utils

import (
	"strconv"
)

// HexGidToInt64 converts a 16-character hex string (Aria2 GID) to int64.
// Note: Aria2 GID is 64-bit. Hex string is 16 chars.
// If the value exceeds int64 positive range (first bit 1), it will wrap to negative
// or parse error if strict positive.
// ParseInt handles negative hex? No, base 16 assumes unsigned unless prefix?
// Actually, ParseInt returns int64. If the hex represents a uint64 that is too big, it errors.
// So we should use ParseUint then cast to int64 (which might be negative).
func HexGidToInt64(gid string) int64 {
	u, err := strconv.ParseUint(gid, 16, 64)
	if err != nil {
		// Fallback for non-hex or error (shouldn't happen with valid GIDs)
		return 0
	}
	return int64(u)
}
