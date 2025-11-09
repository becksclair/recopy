package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"recopy/internal/remotepath"
)

type pathResolver struct {
	sources    []string
	dest       string
	destIsDir  bool
	destRemote bool
	remoteSpec remotepath.Spec
}

func newPathResolver(sources []string, dest string) (*pathResolver, error) {
	spec, isRemote, err := remotepath.Parse(dest)
	if err != nil {
		return nil, err
	}
	destIsDir := false
	if isRemote {
		if len(sources) > 1 {
			destIsDir = true
		}
		if strings.HasSuffix(spec.Path, "/") {
			destIsDir = true
		}
		return &pathResolver{
			sources:    append([]string(nil), sources...),
			dest:       dest,
			destIsDir:  destIsDir,
			destRemote: true,
			remoteSpec: spec,
		}, nil
	}
	info, statErr := os.Stat(dest)
	destExists := statErr == nil
	if statErr != nil && !os.IsNotExist(statErr) {
		return nil, statErr
	}
	if destExists {
		destIsDir = info.IsDir()
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
	if r.destRemote {
		spec := r.remoteSpec
		if r.destIsDir {
			base := remoteAwareBase(src)
			spec = remotepath.Join(spec, base)
		}
		return spec.String(), nil
	}
	if r.destIsDir {
		return filepath.Join(r.dest, filepath.Base(src)), nil
	}
	return r.dest, nil
}

func remoteAwareBase(src string) string {
	spec, ok, err := remotepath.Parse(src)
	if err == nil && ok {
		return remotepath.Base(spec)
	}
	return filepath.Base(src)
}
