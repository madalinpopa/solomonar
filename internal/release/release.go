// Package release resolves the release version and stamps it into the embedded assets.
package release

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// Unknown is the version of a build without a release tag in its history.
const Unknown = "0.0.0"

var (
	semver      = regexp.MustCompile(`^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-.+)?$`)
	untagged    = regexp.MustCompile(`^-\d{14}-[0-9a-f]{12}$`)
	afterStable = regexp.MustCompile(`^-0\.\d{14}-[0-9a-f]{12}$`)
)

// Version maps a module build version to the major.minor.patch release it reports (AD-10).
func Version(build string) string {
	build, _, _ = strings.Cut(build, "+")
	m := semver.FindStringSubmatch(build)
	if m == nil || untagged.MatchString(m[4]) {
		return Unknown
	}
	if afterStable.MatchString(m[4]) {
		patch, _ := strconv.Atoi(m[3])
		if patch == 0 {
			return Unknown
		}
		return fmt.Sprintf("%s.%s.%d", m[1], m[2], patch-1)
	}
	return m[1] + "." + m[2] + "." + m[3]
}

// Assets serves fsys with version rendered into the framework and extension Markdown files.
func Assets(fsys fs.FS, version string) fs.FS {
	return stamped{FS: fsys, data: struct{ Version string }{version}}
}

type stamped struct {
	fs.FS
	data struct{ Version string }
}

func (s stamped) Open(name string) (fs.File, error) {
	f, err := s.FS.Open(name)
	if err != nil || !templated(name) {
		return f, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	content, err := s.render(f, name)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	return &file{Reader: bytes.NewReader(content), info: sized{FileInfo: info, size: int64(len(content))}}, nil
}

func (s stamped) render(f fs.File, name string) ([]byte, error) {
	var source bytes.Buffer
	if _, err := source.ReadFrom(f); err != nil {
		return nil, err
	}
	tmpl, err := template.New(name).Option("missingkey=error").Parse(source.String())
	if err != nil {
		return nil, fmt.Errorf("render version: %w", err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, s.data); err != nil {
		return nil, fmt.Errorf("render version: %w", err)
	}
	return out.Bytes(), nil
}

// templated reports whether name carries the version; templates/ is rendered later with project data.
func templated(name string) bool {
	return path.Ext(name) == ".md" && (strings.HasPrefix(name, "framework/") || strings.HasPrefix(name, "extensions/"))
}

type file struct {
	*bytes.Reader
	info fs.FileInfo
}

func (f *file) Stat() (fs.FileInfo, error) { return f.info, nil }

func (*file) Close() error { return nil }

type sized struct {
	fs.FileInfo
	size int64
}

func (s sized) Size() int64 { return s.size }
