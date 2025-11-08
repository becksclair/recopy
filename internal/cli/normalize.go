package cli

import (
	"fmt"
	"path/filepath"
	"strings"
)

func normalizePaths(sources []string, dest string) ([]string, string, error) {
	cleanedSources := make([]string, len(sources))
	absSources := make([]string, len(sources))
	for i, src := range sources {
		if src == "" {
			return nil, "", fmt.Errorf("source path %d is empty", i)
		}
		cleaned := filepath.Clean(src)
		cleanedSources[i] = cleaned
		absSrc, err := filepath.Abs(cleaned)
		if err != nil {
			return nil, "", fmt.Errorf("resolve source %q: %w", src, err)
		}
		absSources[i] = absSrc
	}

	if dest == "" {
		return nil, "", fmt.Errorf("destination path is empty")
	}
	cleanedDest := filepath.Clean(dest)
	absDest, err := filepath.Abs(cleanedDest)
	if err != nil {
		return nil, "", fmt.Errorf("resolve destination %q: %w", dest, err)
	}

	sep := string(filepath.Separator)
	for i, absSrc := range absSources {
		if absDest == absSrc || strings.HasPrefix(absDest, absSrc+sep) {
			return nil, "", fmt.Errorf("destination %q is inside source %q", cleanedDest, cleanedSources[i])
		}
	}

	return cleanedSources, cleanedDest, nil
}
