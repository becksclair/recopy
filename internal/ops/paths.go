package ops

import (
	"fmt"
	"os"
	"path/filepath"
)

type pathResolver struct {
	sources   []string
	dest      string
	destIsDir bool
}

func newPathResolver(sources []string, dest string) (*pathResolver, error) {
	info, err := os.Stat(dest)
	destExists := err == nil
	destIsDir := destExists && info.IsDir()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if len(sources) > 1 && !destIsDir {
		return nil, fmt.Errorf("destination %s must be a directory when copying multiple sources", dest)
	}
	if len(sources) > 1 {
		destIsDir = true
	}
	return &pathResolver{sources: append([]string(nil), sources...), dest: dest, destIsDir: destIsDir}, nil
}

func (r *pathResolver) TargetFor(src string) (string, error) {
	if r.destIsDir {
		return filepath.Join(r.dest, filepath.Base(src)), nil
	}
	return r.dest, nil
}
