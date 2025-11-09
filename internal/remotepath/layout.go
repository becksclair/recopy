package remotepath

import "fmt"

// Side enumerates which half of the transfer is remote.
type Side int

const (
	SideNone Side = iota
	SideSource
	SideDest
)

// Layout summarizes remote participation for a copy/move run.
type Layout struct {
	Side Side
	Spec Spec // host metadata (Path ignored)
}

// ClassifyPaths enforces remote path rules and records which side is remote.
func ClassifyPaths(sources []string, dest string) (Layout, error) {
	var remoteSrc Specs
	for _, src := range sources {
		spec, ok, err := Parse(src)
		if err != nil {
			return Layout{}, err
		}
		if ok {
			remoteSrc = append(remoteSrc, spec)
		}
	}
	destSpec, destRemote, err := Parse(dest)
	if err != nil {
		return Layout{}, err
	}
	if destRemote && len(remoteSrc) > 0 {
		return Layout{}, fmt.Errorf("remote sources and remote destination together not supported")
	}
	if len(remoteSrc) > 0 {
		if len(remoteSrc) != len(sources) {
			return Layout{}, fmt.Errorf("cannot mix remote and local sources")
		}
		first := remoteSrc[0]
		for _, spec := range remoteSrc[1:] {
			if spec.Host != first.Host || spec.User != first.User {
				return Layout{}, fmt.Errorf("remote sources must share the same login")
			}
		}
		first.Path = ""
		return Layout{Side: SideSource, Spec: first}, nil
	}
	if destRemote {
		destSpec.Path = ""
		return Layout{Side: SideDest, Spec: destSpec}, nil
	}
	return Layout{Side: SideNone}, nil
}

// Specs is a helper slice type.
type Specs []Spec
