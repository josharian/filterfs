// Package filterfs provides fs.FS wrappers that filter filesystems.
// Filters can include or exclude only specific files.
package filterfs

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// ExcludePaths returns a filesystem identical to fsys excluding all paths in hide.
// Hiding a directory hides all contained subdirectories and files.
// ExcludePaths panics if hide includes the path ".".
func ExcludePaths(fsys fs.FS, hide ...string) fs.FS {
	hidden := make(map[string]bool)
	for _, path := range hide {
		if path == "." {
			panic(`ExcludePaths: cannot hide path "."`)
		}
		hidden[path] = true
	}
	return ExcludeFn(fsys, func(s string) bool { return hidden[s] })
}

// pathPrefixes returns all prefixes of path that represent files (or dirs).
// pathPrefixes("a/b/c") returns "a/b/c", "a/b", and "a".
// TODO: come up with a readable way to do this that avoids allocating a slice
// TODO: once that exists, propose as an addition to path/filepath, 'cause I end up needing this an awful lot
func pathPrefixes(path string) []string {
	path = filepath.Clean(path)
	pfxs := make([]string, 0, strings.Count(path, "/"))
	pfxs = append(pfxs, path)
	for {
		path = filepath.Dir(path)
		pfxs = append(pfxs, path)
		if path == "." || path == "/" {
			return pfxs
		}
	}
}

// ExcludeFn returns a filesystem identical to fsys excluding paths for which hide(path) returns true.
// Hiding a directory hides all contained subdirectories and files.
// ExcludeFn panics if hide(".") returns true.
func ExcludeFn(fsys fs.FS, hide func(string) bool) fs.FS {
	if hide(".") {
		panic(`ExcludeFn: cannot hide path "."`)
	}
	return &excludeFnFS{fsys: fsys, hide: hide}
}

type excludeFnFS struct {
	fsys fs.FS
	hide func(string) bool
}

func (f *excludeFnFS) Open(name string) (fs.File, error) {
	pfxs := pathPrefixes(name)
	for _, pfx := range pfxs {
		if f.hide(pfx) {
			return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
		}
	}
	file, err := f.fsys.Open(name)
	if err != nil {
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		if rdf, ok := file.(fs.ReadDirFile); ok {
			return &excludeFnDir{path: name, excludeFnFS: f, ReadDirFile: rdf}, nil
		}
	}
	return file, nil
}

type excludeFnDir struct {
	path string
	*excludeFnFS
	fs.ReadDirFile
}

func (f *excludeFnDir) ReadDir(n int) ([]fs.DirEntry, error) {
	des, err := f.ReadDirFile.ReadDir(n)
	if err != nil {
		return nil, err
	}

	var dst int
	for _, de := range des {
		path := filepath.Clean(f.path + "/" + de.Name())
		if f.hide(path) {
			continue
		}
		des[dst] = de
		dst++
	}

	tail := des[dst:]
	des = des[:dst]
	for i := range tail {
		tail[i] = nil
	}

	return des, nil
}

// TODO: Add other extension methods.
// These require extra care to ensure that hiding a directory
// also hides all contained subdirectories and files.

// func (f *excludeFnFS) Glob(pattern string) ([]string, error)
// func (f *excludeFnFS) ReadFile(name string) ([]byte, error)
// func (f *excludeFnFS) Stat(name string) (fs.FileInfo, error)
// func (f *excludeFnFS) Sub(dir string) (fs.FS, error)
