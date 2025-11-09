package remotepath

import (
	"fmt"
	"path"
	"strings"
)

// Spec represents a parsed rsync/ssh-style remote path (user@host:/path).
type Spec struct {
	User string
	Host string
	Path string
}

// String reconstructs the rsync-style path.
func (s Spec) String() string {
	addr := s.Host
	if s.User != "" {
		addr = s.User + "@" + addr
	}
	return addr + ":" + s.Path
}

// Address returns the user@host portion (user optional).
func (s Spec) Address() string {
	if s.User == "" {
		return s.Host
	}
	return s.User + "@" + s.Host
}

// Parse attempts to interpret raw as a remote path. ok=false means not remote.
func Parse(raw string) (Spec, bool, error) {
	hostPart, pathPart, ok := splitHostPath(raw)
	if !ok {
		return Spec{}, false, nil
	}
	if pathPart == "" {
		return Spec{}, false, fmt.Errorf("remote path %q missing path component", raw)
	}
	if looksLikeDrive(hostPart) || strings.ContainsAny(hostPart, "/\\") {
		return Spec{}, false, nil
	}
	user := ""
	host := hostPart
	if at := strings.LastIndex(hostPart, "@"); at >= 0 {
		user = hostPart[:at]
		host = hostPart[at+1:]
	}
	if host == "" {
		return Spec{}, false, fmt.Errorf("remote path %q missing host", raw)
	}
	return Spec{User: user, Host: host, Path: pathPart}, true, nil
}

// IsRemote reports whether raw matches remote syntax.
func IsRemote(raw string) bool {
	_, ok, _ := Parse(raw)
	return ok
}

// Join appends elem to the Spec path using POSIX semantics.
func Join(spec Spec, elem string) Spec {
	if spec.Path == "" {
		spec.Path = elem
		return spec
	}
	base := spec.Path
	if strings.HasSuffix(base, "/") {
		base = strings.TrimSuffix(base, "/")
	}
	spec.Path = path.Join(base, elem)
	return spec
}

// Base returns the final element of the Spec path.
func Base(spec Spec) string {
	if spec.Path == "" {
		return ""
	}
	trimmed := strings.TrimSuffix(spec.Path, "/")
	if trimmed == "" {
		return "/"
	}
	return path.Base(trimmed)
}

func splitHostPath(raw string) (string, string, bool) {
	inBracket := false
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '[':
			inBracket = true
		case ']':
			inBracket = false
		case ':':
			if inBracket {
				continue
			}
			return raw[:i], raw[i+1:], true
		}
	}
	return "", "", false
}

func looksLikeDrive(head string) bool {
	if len(head) != 1 {
		return false
	}
	ch := head[0]
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}
