package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"recopy/internal/remotepath"
)

func normalizePaths(sources []string, dest string) ([]string, string, error) {
	cleanedSources := make([]string, len(sources))
	absSources := make([]string, len(sources))
	for i, src := range sources {
		if src == "" {
			return nil, "", fmt.Errorf("source path %d is empty", i)
		}
		spec, remote, err := remotepath.Parse(src)
		if err != nil {
			return nil, "", err
		}
		if remote {
			cleanedSources[i] = spec.String()
			absSources[i] = ""
			continue
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
	spec, destRemote, err := remotepath.Parse(dest)
	if err != nil {
		return nil, "", err
	}
	cleanedDest := dest
	absDest := ""
	if destRemote {
		cleanedDest = spec.String()
	} else {
		cleanedDest = filepath.Clean(dest)
		absDest, err = filepath.Abs(cleanedDest)
		if err != nil {
			return nil, "", fmt.Errorf("resolve destination %q: %w", dest, err)
		}
	}

	sep := string(filepath.Separator)
	for i, absSrc := range absSources {
		if absSrc == "" || absDest == "" {
			continue
		}
		if absDest == absSrc || strings.HasPrefix(absDest, absSrc+sep) {
			return nil, "", fmt.Errorf("destination %q is inside source %q", cleanedDest, cleanedSources[i])
		}
	}

	return cleanedSources, cleanedDest, nil
}
