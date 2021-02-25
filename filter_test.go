package filterfs

import (
	"errors"
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestExcludePaths(t *testing.T) {
	mapfs := fstest.MapFS{
		"a":     &fstest.MapFile{Data: []byte{'a'}},
		"b/c":   &fstest.MapFile{},
		"d":     &fstest.MapFile{Data: []byte{'d'}},
		"e/f":   &fstest.MapFile{Data: []byte{'f'}},
		"g/h/i": &fstest.MapFile{Data: []byte{'i'}},
	}

	hfs := ExcludePaths(mapfs, "b", "f", "g/h")
	err := fstest.TestFS(hfs, "a", "d", "e", "g")
	if err != nil {
		t.Error(err)
	}

	for _, exclude := range []string{"b", "f", "b/c", "g/h", "g/h/i"} {
		_, err := fs.Stat(hfs, exclude)
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("fsys contains excluded file %s", exclude)
		}
	}
}

func TestPathPrefixes(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{in: "a/b/c", want: []string{"a/b/c", "a/b", "a", "."}},
	}

	for _, tt := range tests {
		got := pathPrefixes(tt.in)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("pathPrefixes(%q) = %v want %v", tt.in, got, tt.want)
		}
	}
}
