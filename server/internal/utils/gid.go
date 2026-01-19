package utils

import (
	"strconv"
)

// HexGidToInt64 converts a 16-character hex string (Aria2 GID) to a positive int64.
// Note: Aria2 GID is 64-bit. We mask the high bit to ensure the result is positive,
// which is required by some Rclone RC implementations for job IDs.
func HexGidToInt64(gid string) int64 {
	u, err := strconv.ParseUint(gid, 16, 64)
	if err != nil {
		return 0
	}
	// Mask high bit to ensure positive int64
	return int64(u & 0x7FFFFFFFFFFFFFFF)
}
