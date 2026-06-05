package storage

import "testing"

// FuzzSanitizeFilename fuzzes sanitizeFilename with arbitrary strings.
func FuzzSanitizeFilename(f *testing.F) {
	seeds := []string{
		"../../etc/passwd",
		"../relative/path",
		"normal-file.txt",
		"file with spaces.pdf",
		"UPPERCASE.JPG",
		"a",
		"",
		"path/with/slashes",
		"special!@#$%^&*()",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, name string) {
		safe := sanitizeFilename(name)
		// Must not contain path separators
		for _, r := range safe {
			if r == '/' || r == '\\' {
				t.Errorf("sanitizeFilename(%q) = %q contains path separator", name, safe)
			}
		}
	})
}
