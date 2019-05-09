package storage

import (
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var store Storer

func MakeFile(name string, content []byte) *File {
	if content == nil {
		content = []byte{}
	}
	return &File{name: name, content: content}
}

type File struct {
	content []byte
	name    string
}

func (w *File) UTime(string) {}

func (w *File) IsCompressed() bool {
	return false
}

func (w *File) Name() string {
	return w.name
}

func (w *File) Path(t *Dir) string {
	return path.Join(t.Path(), w.name)
}

func (w *File) Write(b []byte) error {
	if b == nil {
		b = []byte{}
	}
	w.content = b
	return nil
}

func (w *File) Read() ([]byte, error) {
	return w.content, nil
}

func TestStore(t *testing.T) {
	t.Run("Lock tests", func(t *testing.T) {
		t.Run("Lock a Name", func(t *testing.T) {
			err := store.Lock("key3", "c1")
			assert.Nil(t, err)
		})

		t.Run("Un-Idempotent Lock a Name", func(t *testing.T) {
			err := store.Lock("key3", "c12")
			assert.NotNil(t, err, "Should have raised a key")
		})

		t.Run("Release a Name", func(t *testing.T) {
			err := store.Unlock("key3")
			assert.Nil(t, err)
		})

		t.Run("Idempotent Release a Name", func(t *testing.T) {
			err := store.Unlock("key3")
			assert.Nil(t, err)
		})
	})

	t.Run("Storage tests", func(t *testing.T) {
		tree := MakeDir("store_tree")

		wid := fmt.Sprintf("alibaba-%s", GenerateUuid())
		workspace := MakeFile(wid, nil)

		t.Run("Workspace does not exist", func(t *testing.T) {
			err := store.Get(workspace, tree)
			assert.Nil(t, err, "Should be nil")
		})

		t.Run("Get a Workspace after creation", func(t *testing.T) {
			err := store.Save(workspace, tree)
			assert.Nil(t, err)

			err = store.Get(workspace, tree)
			assert.Nil(t, err)
		})

		t.Run("Re-saving a Workspace doesn't raise an Error", func(t *testing.T) {
			err := store.Save(workspace, tree)
			assert.Nil(t, err)

			err = store.Get(workspace, tree)
			assert.Nil(t, err)

			v, err := store.GetVersions(workspace, tree)
			assert.Nil(t, err)

			assert.Equal(t, 3, len(v))
			assert.Contains(t, strings.Join(v, ""), "latest")
		})

		t.Run("Get absent Name", func(t *testing.T) {
			w := MakeFile("hello/world", nil)

			err := store.Get(w, nil)
			assert.Nil(t, err)
			assert.Equal(t, []byte{}, w.content)
		})

		t.Run("Get Valid Name", func(t *testing.T) {
			key := fmt.Sprintf("workspaces/%v/latest", wid)
			w := MakeFile(key, nil)
			err := store.Get(w, nil)

			assert.Nil(t, err)
			assert.Equal(t, []byte{}, w.content)
		})

		t.Run("Get Keys", func(t *testing.T) {
			prefix := "workspaces/"
			separator := "/"

			keys, err := store.GetKeys(prefix, separator)
			assert.Nil(t, err)

			for _, k := range keys {
				splits := strings.Split(k, "/")
				assert.Equal(t, 3, len(splits))
			}
		})

		t.Run("Save and Get Name", func(t *testing.T) {
			key := GenerateUuid()
			val := GenerateUuid()

			w := MakeFile(key, []byte(val))
			err := store.Save(w, nil)

			assert.Nil(t, err)

			gerr := store.Get(w, nil)
			assert.Nil(t, gerr)
			assert.Equal(t, val, string(w.content))
		})
	})
}

func TestTree(t *testing.T) {
	t.Run("Test Trees", func(t *testing.T) {
		t.Run("Empty Dir", func(t *testing.T) {
			x := MakeDir()
			assert.Nil(t, x, "Should be nil")
		})

		t.Run("One node", func(t *testing.T) {
			x := MakeDir("a")
			assert.Equal(t, x.Path(), "a")
		})

		t.Run("Even node", func(t *testing.T) {
			x := MakeDir("a", "b")
			assert.Equal(t, x.Path(), "a/b")
		})

		t.Run("Odd Nodes", func(t *testing.T) {
			x := MakeDir("a", "b", "c")
			assert.Equal(t, x.Path(), "a/b/c")
		})
	})
}