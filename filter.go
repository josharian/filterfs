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
	return &excludePathsFS{fsys: fsys, hidden: hidden}
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

type excludePathsFS struct {
	fsys   fs.FS
	hidden map[string]bool
}

func (f *excludePathsFS) Open(name string) (fs.File, error) {
	pfxs := pathPrefixes(name)
	for _, pfx := range pfxs {
		if f.hidden[pfx] {
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
			return &excludePathsDir{path: name, excludePathsFS: f, ReadDirFile: rdf}, nil
		}
	}
	return file, nil
}

type excludePathsDir struct {
	path string
	*excludePathsFS
	fs.ReadDirFile
}

func (f *excludePathsDir) ReadDir(n int) ([]fs.DirEntry, error) {
	des, err := f.ReadDirFile.ReadDir(n)
	if err != nil {
		return nil, err
	}

	var dst int
	for _, de := range des {
		path := filepath.Join(f.path, de.Name())
		if f.hidden[path] {
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

// func (f *excludePathsFS) Glob(pattern string) ([]string, error)
// func (f *excludePathsFS) ReadFile(name string) ([]byte, error)
// func (f *excludePathsFS) Stat(name string) (fs.FileInfo, error)
// func (f *excludePathsFS) Sub(dir string) (fs.FS, error)
