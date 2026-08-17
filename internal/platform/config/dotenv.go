package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// loadDotEnv reads KEY=VALUE lines from path into the process environment.
// Variables already present in the real environment always win and are never
// overwritten, so .env only fills gaps: production and docker-compose env vars
// keep precedence over a checked-in-less .env file. A missing file is not an
// error (all keys fall back to their compiled defaults).
func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d: expected KEY=VALUE, got %q", path, lineNo, line)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("%s:%d: empty key", path, lineNo)
		}
		value = strings.TrimSpace(value)
		// A value wrapped in matching quotes keeps its contents verbatim
		// (including #, spaces); otherwise a trailing # starts a comment.
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') {
			if q := value[0]; value[len(value)-1] == q {
				value = value[1 : len(value)-1]
			}
		} else if i := strings.IndexByte(value, '#'); i >= 0 {
			value = strings.TrimSpace(value[:i])
		}

		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
	return sc.Err()
}
