package names

import (
	"fmt"
	"hash/adler32"
	"hash/fnv"
)

// FNVName formats `<prefix>-<checksum>` using the FNV-1a 32-bit hash algorithm.
func FNVName(prefix, name string) string {
	h := fnv.New32a()
	h.Write([]byte(name))
	return fmt.Sprintf("%s-%x", prefix, h.Sum32())
}

// Adler32Name formats `<prefix>-<checksum>` and matches Helm's built-in adler32 helper.
// Useful when we need deterministic names in templates but only adler32 is available.
func Adler32Name(prefix, name string) string {
	hash := Adler32Checksum(name)
	return fmt.Sprintf("%s-%s", prefix, hash)
}

// Adler32Checksum returns the decimal adler32 checksum of the provided name.
// Matches Helm's `adler32sum` helper so that templates stay consistent.
func Adler32Checksum(name string) string {
	return fmt.Sprintf("%d", adler32.Checksum([]byte(name)))
}
