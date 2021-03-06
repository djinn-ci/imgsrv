package main

import (
	"os"
	"time"
)

type Image struct {
	Path     string    `json:"-"`
	Driver   string    `json:"driver"`
	Category string    `json:"category"`
	Group    string    `json:"group"`
	Name     string    `json:"name"`
	Link     string    `json:"link"`
	ModTime  time.Time `json:"mod_time"`
}

func (i *Image) Data() (ReadSeekCloser, error) {
	return os.Open(i.Path)
}

func (i *Image) Endpoint() string {
	s := "/" + i.Driver

	if i.Category != "" {
		s += "/" + i.Category
	}
	return s + "/" + i.Name
}

// Tree offers a hierarchical way of organizing images by their driver, category
// and group. The driver, category, and group of an image are used as the key
// under which an image would be stored, if they are not empty.
//
// Each time an image is stored in a tree, a new sub-tree may be created under
// which the image is actually stored. For example, if an image has the driver
// of "qemu", but not category or group, then it will be stored under a tree
// named "qemu". If an image has no driver, category or group, then it is
// stored in the root tree, that is, the top-level.
type Tree struct {
	name     string
	order    []string
	images   []*Image
	children map[string]*Tree
}

func (t *Tree) get(key string) *Tree {
	if t.children == nil {
		t.children = make(map[string]*Tree)
	}

	t2, ok := t.children[key]

	if !ok {
		t.order = append(t.order, key)
		t2 = &Tree{name: key}
		t.children[key] = t2
	}
	return t2
}

func (t *Tree) put(img *Image) {
	t.images = append(t.images, img)
}

func (t *Tree) Put(img *Image) {
	root := t

	for _, key := range []string{img.Driver, img.Category, img.Group} {
		if key == "" {
			break
		}
		root = root.get(key)
	}
	root.put(img)
}

func (t *Tree) Walk(visit func(string, []*Image)) {
	visit(t.name, t.images)

	for _, key := range t.order {
		if t2, ok := t.children[key]; ok {
			t2.Walk(visit)
		}
	}
}

func (t *Tree) Name() string { return t.name }

func (t *Tree) Images() []*Image { return t.images }

func (t *Tree) Children() []*Tree {
	tt := make([]*Tree, 0, len(t.children))

	for _, name := range t.order {
		tt = append(tt, t.children[name])
	}
	return tt
}

func (t *Tree) HasChildren() bool { return len(t.children) > 0 }
