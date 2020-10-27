package store

import (
	"testing"
)

var (
	root *TreeNode
)

func TestTree1(t *testing.T) {
	CreateDB("dbTest")

	// the following is duplicated verbatim from func makeTestDB
	db.Exec("DELETE FROM chunks")

	data1 := []byte("very nice")
	data2 := []byte("mediocre")
	data3 := []byte("")
	sig1 := computeSig(data1)
	sig2 := computeSig(data2)
	sig3 := computeSig(data3)
	db.Exec("INSERT INTO chunks (sig, data) VALUES (?,?)", sig1, data1)
	db.Exec("INSERT INTO chunks (sig, data) VALUES (?,?)", sig2, data2)
	db.Exec("INSERT INTO chunks (sig, data) VALUES (?,?)", sig3, data3)

	// Resulting chunks are:
	// sha256_32_GQ3BJPG3QKPX4CVR4OUHNXCUG6QBU3XKHXZ72EBRO5FV42I73FEQ====
	// sha256_32_GKR55CZCIXM62KB4UKGXFJKZKKNZC6I5JOJGUDQGZ6K7DW466ZDA====
	// sha256_32_4OYMIQUY7QOBJGX36TEJS35ZEQT24QPEMSNZGTFESWMRW6CSXBKQ====
	//
	// For reference: 	encodeStd    = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

	root = createTree(1)
	if root == nil {
		t.Fatalf("tree1 err: %v\n", err)
	}

	if num := len(root.children); num != 0 {
		t.Fatalf("tree1 err #children: %v\n", num)
	}

	if num := len(root.ChunkSigs); num != 3 {
		t.Fatalf("tree1 A err #sigs: %v\n", num)
	}

}

func TestTree2(t *testing.T) {
	root = createTree(2)
	if root == nil {
		t.Fatalf("tree1 err: %v\n", err)
	}

	if num := len(root.children); num != NUM_CHILDREN {
		t.Fatalf("tree1 err #children: %v\n", num)
	}

	if num := len(root.children[6].ChunkSigs); num != 2 {
		t.Fatalf("tree1 A err #sigs: %v\n", num)
	}

}

func TestTree3(t *testing.T) {
	root = createTree(3)
	if root == nil {
		t.Fatalf("tree1 err: %v\n", err)
	}

	if num := len(root.children); num != NUM_CHILDREN {
		t.Fatalf("tree3 root #children: %v\n", num)
	}

	if num := len(root.children[0].children); num != NUM_CHILDREN {
		t.Fatalf("tree3 level 2 #children: %v\n", num)
	}

	if num := len(root.children[6].children[10].ChunkSigs); num != 1 {
		t.Fatalf("tree3 G - K #sigs: %v\n", num)
	}

}
