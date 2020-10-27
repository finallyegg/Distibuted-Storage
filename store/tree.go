package store

import (
	"crypto/sha256"
	"encoding/base32"
	"sort"
	. "strings"
)

const (
	NUM_CHILDREN = 32
	encodeStd    = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
)

type TreeNode struct {
	Sig       string
	ChildSigs []string
	ChunkSigs []string
	children  []*TreeNode
}

var (
	reverseEncode = map[string]int{}
	treeRoots     = map[string]*TreeNode{}
)

// ConstructTree genertate tree skeloton
func ConstructTree(root *TreeNode, currentHeight int, Height int) {
	if currentHeight == Height {
		return
	}
	for i := 0; i < 32; i++ {
		var newNode TreeNode
		ConstructTree(&newNode, currentHeight+1, Height)
		root.children = append(root.children, &newNode)
	}

}

// get array of node index for later on insertation
func getEncodeIndexSequence(sig string, Height int) []int {

	// construct reverseEncode Map
	for i, c := range Split(encodeStd, "") {
		reverseEncode[c] = i
	}

	var ret []int
	sigNoPrefix := sig[10:]

	// get inseration order based on current height
	for i := 0; i < Height-1; i++ {
		ret = append(ret, reverseEncode[string(sigNoPrefix[i])])
	}
	return ret
}

// insertIntoTree insert sig into tree based on tree's height
func insertIntoTree(root *TreeNode, Height int, sig string) *TreeNode {
	seq := getEncodeIndexSequence(sig, Height)

	node := root
	// treversal node to insertion point
	for _, v := range seq {
		node = node.children[v]
	}
	node.ChunkSigs = append(node.ChunkSigs, sig)
	return node
}

func generateTreeHash(treeMap map[string]*TreeNode, node *TreeNode) {
	if len(node.children) == 0 { // leaf node

		var allChunkSigs string
		chunkSigsCopy := node.ChunkSigs[:]
		sort.Strings(chunkSigsCopy)
		for _, v := range chunkSigsCopy {
			allChunkSigs += v
		}
		data := []byte(allChunkSigs)
		sh := sha256.Sum256(data)
		node.Sig = "sha256_32_" + base32.StdEncoding.EncodeToString(sh[:])
		treeMap[node.Sig] = node
	} else {
		for i := 0; i < 32; i++ {
			generateTreeHash(treeMap, node.children[i])
			node.ChildSigs = append(node.ChildSigs, node.children[i].Sig)
		}
		childSigsCopy := node.ChildSigs[:]
		sort.Strings(childSigsCopy)

		var allChildSigs string
		for i := 0; i < 32; i++ {
			allChildSigs += childSigsCopy[i]
			data := []byte(allChildSigs)
			sh := sha256.Sum256(data)
			node.Sig = "sha256_32_" + base32.StdEncoding.EncodeToString(sh[:])
			treeMap[node.Sig] = node
		}
	}
}

// FillTree fill tree with all hash value and return a map of sig -> node
func FillTree(DBpath string, root *TreeNode, Height int) map[string]*TreeNode {
	recieptMap := make(map[string]*TreeNode)
	data := GetAllFromDB(DBpath) // get all data hash
	for _, v := range data {
		insertIntoTree(root, Height, v)
	}
	generateTreeHash(recieptMap, root)
	return recieptMap
}
