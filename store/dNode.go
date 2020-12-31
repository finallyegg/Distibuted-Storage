package store

import "bazil.org/fuse"

type DNode2 struct {
	Name       string
	Attrs      fuse.Attr
	ParentSig  string
	Version    int
	PrevSig    string
	ChildSigs  map[string]string
	DataBlocks []string //sigs

	sig       string
	dirty     bool
	metaDirty bool
	expanded  bool
	parent    *DNode2
	children  map[string]*DNode2
	data      []byte
}
