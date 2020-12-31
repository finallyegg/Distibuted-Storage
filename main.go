// memfs implements a simple in-memory file system.
package main

/*
 Two main files are ../fuse.go and ../fs/serve.go
*/

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	. "github.com/mattn/go-getopt"

	. "gitlab.cs.umd.edu/cmsc818eFall20/cmsc818e-zepinghe/p5b/rkChunk"
	. "gitlab.cs.umd.edu/cmsc818eFall20/cmsc818e-zepinghe/p5b/store"

	"golang.org/x/net/context"
)

/*
    Need to implement these types from bazil/fuse/fs!

    type FS interface {
	  // Root is called to obtain the Node for the file system root.
	  Root() (Node, error)
    }

    type Node interface {
	  // Attr fills attr with the standard metadata for the node.
	  Attr(ctx context.Context, attr *fuse.Attr) error
    }
*/

type FS interface {
	// Root is called to obtain the Node for the file system root.
	Root() (Node, error)
}

type Node interface {
	// Attr fills attr with the standard metadata for the node.
	Attr(ctx context.Context, attr *fuse.Attr) error
}

//=============================================================================

type Head struct {
	Root    string // current Root signature
	NextInd uint64 // Next AVAILABLE inode number
	Replica uint64 // Ignore for now
}

type Dfs struct{}

type status struct {
	readNodes     int
	expansions    int
	byteWritten   int
	serverRequest int
	byteSent      int
}

var (
	root       *DNode
	Debug      = false
	printCalls = true
	conn       *fuse.Conn
	mountPoint = "dss"        // default mount point
	uid        = os.Geteuid() // use the same uid/gid for all
	gid        = os.Getegid()

	servAddr    = ""
	flushPeriod = 5
	reset       = false
	timeTravel  = ""
	head        = Head{NextInd: uint64(rand.Uint64()), Root: ""}

	chFlush chan bool = nil

	currentStat status

	_ fs.Node = (*DNode)(nil) // make sure that DNode implements bazil's Node interface
	_ fs.FS   = (*Dfs)(nil)   // make sure that DNode implements bazil's FS interface
)

//=============================================================================
type DNode struct {
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
	parent    *DNode
	children  map[string]*DNode
	data      []byte
}

// Implement:
func (n *DNode) init(name string, mode os.FileMode, expanded bool) error {
	// fmt.Println("called init")
	// <-chFlush
	n.Name = name
	n.Version = 1
	n.PrevSig = ""
	n.dirty = false
	n.Attrs.Mode = mode
	n.ChildSigs = make(map[string]string)
	n.children = make(map[string]*DNode)
	n.Attrs.Ctime = time.Now()
	n.Attrs.Crtime = time.Now()
	n.Attrs.Atime = time.Now()
	n.Attrs.Uid = uint32(uid)
	n.Attrs.Gid = uint32(gid)
	n.Attrs.Inode = uint64(rand.Uint64())
	n.expanded = expanded
	// chFlush <- true
	return nil
}

func (Dfs) Root() (n fs.Node, err error) {
	// <-chFlush
	fmt.Println("ROOT")
	// chFlush <- true
	return root, nil
}

func (n *DNode) Attr(ctx context.Context, attr *fuse.Attr) error {
	// fmt.Println("Attr", " for ", n.Name)
	// <-chFlush
	*attr = n.Attrs
	// chFlush <- true
	return nil
}

func (n *DNode) Lookup(ctx context.Context, name string) (fs.Node, error) {
	<-chFlush
	switch name {
	case "mach_kernel", ".hidden", "._.":
		// Just quiet some log noise on OS X.
		chFlush <- true
		return nil, fuse.ENOENT
	}
	n.expand()
	node, found := n.children[name]

	if found {
		chFlush <- true
		return node, nil
	} else {
		// fmt.Println("Not Found", name, "in", n.Name)
		chFlush <- true
		return nil, fuse.ENOENT
	}

}

func (n *DNode) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	// fmt.Println("READ all in ", n.Name)
	<-chFlush
	if !n.expanded {
		n.expand()
	}

	var files []fuse.Dirent
	for _, nodes := range n.children {
		var dirent fuse.Dirent
		if nodes.Attrs.Mode.IsDir() {
			dirent = fuse.Dirent{
				Name: nodes.Name,
				Type: fuse.DT_Dir,
			}
		} else {
			dirent = fuse.Dirent{
				Name: nodes.Name,
				Type: fuse.DT_File,
			}
		}
		files = append(files, dirent)
	}
	chFlush <- true
	if len(files) == 0 {
		return nil, fuse.ENOENT
	}
	currentStat.readNodes++
	return files, nil
}

func (n *DNode) Getattr(ctx context.Context, req *fuse.GetattrRequest, resp *fuse.GetattrResponse) error {
	// fmt.Println("Get attr", n.Name)
	resp.Attr = n.Attrs
	return nil
}

func (p *DNode) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	<-chFlush
	fmt.Println("Mkdir", req.Name, " in ", p.Name)
	dirName := req.Name
	newDir := new(DNode)
	newDir.init(dirName, os.ModeDir|0766, true)
	newDir.Attrs.Inode = uint64(rand.Uint64())
	newDir.parent = p
	newDir.ParentSig = p.sig

	// sig := sendDir(servAddr, newDir)
	// newDir.sig = sig

	// p.ChildSigs[dirName] = sig
	p.children[dirName] = newDir
	p.Attrs.Size++
	p.Attrs.Ctime = time.Now()
	p.Attrs.Inode = uint64(rand.Uint64())

	nameLength := len(newDir.Name)
	chFlush <- true
	if nameLength > 10 && newDir.Name[nameLength-10:nameLength] == "@@versions" {
		trueName := newDir.Name[:nameLength-10]
		fileNode, err := p.Lookup(ctx, trueName)
		if err != nil {
			fmt.Println("NO FILE", trueName)
		}
		buf := fuse.Attr{}
		fileNode.Attr(ctx, &buf)

		versionList := GetAllVersions(buf.Inode)
		for _, sig := range versionList {
			newNode := new(DNode)
			newNode.init("", os.ModeDir|0444, false)

			getDNode(newNode, servAddr, sig)
			newNode.parent = newDir
			newNode.Name = newNode.Name + "." + newNode.Attrs.Mtime.Format("2006-01-02 15:04:05")
			if newNode.Attrs.Mode.IsDir() {
				newNode.Attrs.Mode = os.ModeDir | 444
			} else {
				newNode.Attrs.Mode = 0400
			}
			newDir.children[newNode.Name] = newNode
		}
		fmt.Println(trueName, buf.Inode)
	} else {
		setDirty(newDir)
	}

	return newDir, nil
}

func (n *DNode) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	fmt.Println("FSYNC", req.String())
	n.Attrs.Atime = time.Now()
	return nil
}

func (n *DNode) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	// fmt.Println("Set attr", "name", n.Name)
	n.Attrs.Ctime = time.Now()
	n.Attrs.Atime = time.Now()
	n.Attrs.Inode = uint64(rand.Uint64())

	// n.Attrs.Size = req.Size
	// n.Attrs.Flags = req.Flags
	// n.Attrs.Mode = req.Mode
	resp.Attr = n.Attrs
	return nil
}

func (p *DNode) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	<-chFlush
	fmt.Println("Create", req.Name, " in ", p.Name)
	node := new(DNode)
	node.init(req.Name, req.Mode|0766, true)
	node.parent = p
	node.Attrs.Inode = uint64(rand.Uint64())

	// p.Version++
	p.children[req.Name] = node
	p.Attrs.Size++
	p.Attrs.Ctime = time.Now()
	resp.Attr = node.Attrs
	resp.Generation = 12
	// p.Attrs.Inode = uint64(rand.Uint64())

	setDirty(node)
	chFlush <- true
	fmt.Println("Create Done")
	return node, node, nil
}

func (n *DNode) ReadAll(ctx context.Context) ([]byte, error) {
	// fmt.Println("RAD ALL in", n.Name)
	<-chFlush
	if !n.expanded {
		n.expand()
	}

	chFlush <- true
	currentStat.readNodes++
	return n.data, nil
}

func (n *DNode) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	<-chFlush
	// zero padding
	if len(n.data) < int(req.Offset)+len(req.Data) {
		// fmt.Println("----")
		paddingSize := int(req.Offset) + len(req.Data) - len(n.data)
		padding := make([]uint8, paddingSize)
		n.data = append(n.data, padding...)
	}
	copy(n.data[req.Offset:], req.Data)
	n.Attrs.Atime = time.Now()
	n.Attrs.Mtime = time.Now()
	n.Attrs.Size = uint64(len(n.data))
	resp.Size = int(len(req.Data))
	// n.Attrs.Inode = uint64(rand.Uint64())
	n.dirty = true
	fmt.Println("WRITE in", n.Name, len(req.Data), "@", req.Offset, "now", n.Attrs.Size)
	chFlush <- true
	currentStat.byteWritten += resp.Size
	return nil
}

func (n *DNode) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	<-chFlush
	fmt.Println("FLUSH in", n.Name)
	if len(n.Name) >= 2 && n.Name[:2] == "._" {
		chFlush <- true
		return nil
	}
	n.Attrs.Mtime = time.Now()

	if n.dirty {
		fmt.Println("fileMETA dirty detected!")
		dataSigs := sendFileByte(servAddr, n)
		n.DataBlocks = make([]string, len(dataSigs))
		copy(n.DataBlocks, dataSigs)

		n.parent.children[n.Name] = n
		n.dirty = false
		setDirty(n)
	}

	chFlush <- true
	return nil
}

func (n *DNode) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	name := req.Name
	fmt.Println("REMOVE", name)
	tempNode, found := n.children[name]

	if !found {
		fmt.Println("Not Found ", name, "in node ", n.Name)
		return nil
	}
	if req.Dir {
		for nameKey, nodes := range tempNode.children {
			fmt.Println("---------", nameKey)
			req.Name = nameKey
			req.Gid = uint32(gid)
			req.Uid = uint32(uid)
			req.Dir = nodes.Attrs.Mode.IsDir()
			tempNode.Remove(ctx, req)
		}

	}

	fmt.Println("Delete", n.Name, name)

	delete(n.children, name)
	n.Attrs.Size--
	n.Attrs.Inode = uint64(rand.Uint64())
	n.Attrs.Atime = time.Now()
	return nil
}

func (n *DNode) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	fmt.Println("Rename", req.NewName)
	<-chFlush
	tempNode, found := n.children[req.OldName]
	if !found {
		fmt.Println("not FOund")
		chFlush <- true
		return nil
	}
	temp := new(DNode)
	temp.init(req.NewName, 0755, true)
	copyNode(temp, tempNode)
	temp.parent = n
	// temp.dirty = true
	setDirty(temp)
	n.children[req.NewName] = temp
	// n.Attrs.Inode = uint64(rand.Uint64())

	delete(n.children, req.OldName)
	chFlush <- true
	return nil
}

func (n *DNode) flush2() string {
	if n.children == nil || len(n.children) == 0 {
		n.Attrs.Mtime = time.Now()
		n.sig = sendDNode(n, servAddr)
		n.metaDirty = false
		return n.sig
	} else {
		for childName := range n.children {
			childNode := n.children[childName]
			if childNode.metaDirty {
				childNode.flush2()
			}
			n.ChildSigs[childName] = childNode.sig
		}
		n.Attrs.Mtime = time.Now()
		n.sig = sendDNode(n, servAddr)
		n.metaDirty = false
		return n.sig
	}
}

func (n *DNode) expand() {
	if n.expanded {
		return
	}
	fmt.Println("expanded", n.Name)
	if n.Attrs.Mode.IsDir() {
		for nodeName, nodeSig := range n.ChildSigs {
			newNode := new(DNode)
			newNode.init(nodeName, os.ModeDir|0766, false)
			// newNode
			getDNode(newNode, servAddr, nodeSig)
			if timeTravel != "" {
				newNode.Attrs.Mode = os.FileMode(int(0444))
			}
			newNode.parent = n
			n.children[nodeName] = newNode
		}
	} else {
		for _, nodeSig := range n.DataBlocks {
			req := Message{Version: 1, Type: "get", Sig: nodeSig}
			jsonOutput, _ := json.MarshalIndent(req, "", "    ")
			currentStat.serverRequest++
			currentStat.byteSent += (len(jsonOutput))
			res, err := http.Post(servAddr, "application/json", bytes.NewReader(jsonOutput))
			if res.StatusCode != 200 {
				PrintExit("NOOOO FILE")
			}
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			var m1 = Message{}
			json.Unmarshal(body, &m1)
			n.data = append(n.data, m1.Data...)
		}
	}
	n.expanded = true
	currentStat.expansions++
}

//=====================================================================

func main() {
	var c int
	for {
		if c = Getopt("df:m:ns:t:"); c == EOF {
			break
		}

		switch c {
		case 'd':
			Debug = !Debug
		case 'm':
			mountPoint = OptArg
		case 's':
			servAddr = "http://" + OptArg + "/json"
		case 'f':
			flushPeriod64int, _ := strconv.ParseInt(OptArg, 0, 64)
			flushPeriod = int(flushPeriod64int)
		case 'n':
			reset = !reset
		case 't':
			timeTravel = OptArg
		default:
			println("usage: main.go [-d | -m <mountpt>]", c)
			os.Exit(1)
		}
	}

	root = new(DNode)
	root.init("This_Root", os.ModeDir|0755, true)

	headSig := ""
	if timeTravel != "" {
		headSig = selectHead(timeTravel)
	} else if !reset {
		headSig = GetLastHead(servAddr)
	}

	if headSig != "" {
		getDNode(root, servAddr, headSig)
		root.expanded = false
	}

	PrintDebug("root inode %d\n", int(root.Attrs.Inode))

	if _, err := os.Stat(mountPoint); err != nil {
		os.Mkdir(mountPoint, 0755)
	}
	fuse.Unmount(mountPoint)
	conn, err := fuse.Mount(mountPoint, fuse.FSName("dssFS"), fuse.Subtype("project P5a"),
		fuse.LocalVolume(), fuse.VolumeName("dssFS"))
	if err != nil {
		log.Fatal(err)
	}
	currentStat = status{}

	chFlush = make(chan bool, 1)
	chFlush <- true

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	go func() {
		time.Sleep(time.Second * 5) // wait 5 second at very start
		for {
			<-chFlush
			if root.metaDirty {
				fmt.Println(" -------- FLUSH HAVE BEEN CALLED -------")
				root.flush2()
				claimHead(root, servAddr)
			}

			chFlush <- true
			time.Sleep(time.Second * time.Duration(flushPeriod))
		}
	}()

	go func() {
		<-ch
		fmt.Println()
		fmt.Println(currentStat.readNodes, "nodes read")
		fmt.Println(currentStat.expansions, "expansions")
		fmt.Println(currentStat.byteWritten, "bytes written")
		fmt.Println(currentStat.serverRequest, "server requests")
		fmt.Println(currentStat.byteSent, "bytes sent")
		defer conn.Close()
		fuse.Unmount(mountPoint)
		os.Exit(1)
	}()

	PrintAlways("mt on %q, debug %v\n", mountPoint, Debug)

	err = fs.Serve(conn, Dfs{})
	PrintDebug("AFTER\n")
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-conn.Ready
	if err := conn.MountError; err != nil {
		log.Fatal(err)
	}
}

func PrintAssert(cond bool, s string, args ...interface{}) {
	if !cond {
		PrintExit(s, args...)
	}
}

func PrintDebug(s string, args ...interface{}) {
	if !Debug {
		return
	}
	PrintAlways(s, args...)
}

func PrintExit(s string, args ...interface{}) {
	PrintAlways(s, args...)
	os.Exit(0)
}

func PrintAlways(s string, args ...interface{}) {
	fmt.Printf(s, args...)
}

func PrintCall(s string, args ...interface{}) {
	if printCalls {
		fmt.Printf(s, args...)
	}
}

func sendDir(addr string, node *DNode) string {
	// root.Close()
	dirReceipt := ObjectDir{Version: node.Version, Type: "dir", Name: node.Name, PrevSig: ""}
	jsonOutput, _ := json.MarshalIndent(dirReceipt, "", "    ")
	msg := Message{Version: 1, Type: "put", Data: jsonOutput}
	jsonOutput, _ = json.MarshalIndent(msg, "", "    ")
	currentStat.serverRequest++
	currentStat.byteSent += (len(jsonOutput))
	response, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	body, _ := ioutil.ReadAll(response.Body)
	var respMsg = Message{}
	json.Unmarshal(body, &respMsg)
	return respMsg.Sig
}

func sendFileByte(addr string, node *DNode) []string {
	// read file by maxChunk bytes and store each file data into an array
	lenList := RkMain(node.data)

	var reciept ObjectFile
	reciept.ModTime = time.Now()
	reciept.Name = node.Name
	reciept.Mode = node.Attrs.Mode
	reciept.Type = "file"
	reciept.Version = node.Version

	offset := 0
	for i := 0; i < len(lenList); i++ {
		var message Message

		bufferChunk := node.data[offset : offset+lenList[i]]
		offset += lenList[i]

		message.Sig = ""
		message.Version = 1
		message.Data = bufferChunk
		message.Name = node.Name
		message.Mode = node.Attrs.Mode
		message.ModTime = node.Attrs.Mtime
		message.Type = "put"
		jsonOutput, _ := json.MarshalIndent(message, "", "    ")
		currentStat.serverRequest++
		currentStat.byteSent += (len(jsonOutput))
		response, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		body, _ := ioutil.ReadAll(response.Body)
		var example = Message{}
		json.Unmarshal(body, &example)
		sig := example.Sig
		reciept.Data = append(reciept.Data, sig)
	}
	return reciept.Data

}

func sendDNode(node *DNode, servAddr string) string {
	nodeBuffer, _ := json.Marshal(*node)
	reqMsg := Message{Data: nodeBuffer, Type: "putDNode"}
	reqBuffer, _ := json.Marshal(reqMsg)
	currentStat.serverRequest++
	currentStat.byteSent += (len(reqBuffer))
	servResp, err := http.Post(servAddr, "application/json", bytes.NewReader(reqBuffer))
	// fmt.Println(string(nodeBuffer))
	if err != nil {
		PrintExit("http error", err)
	}
	body, _ := ioutil.ReadAll(servResp.Body)
	var respMsg Message
	json.Unmarshal(body, &respMsg)
	fileSig := respMsg.Sig
	return fileSig
}

func getDNode(dst *DNode, servAddr string, sig string) {
	reqMsg := Message{Type: "get", Sig: sig}
	reqBuffer, _ := json.Marshal(reqMsg)
	currentStat.serverRequest++
	currentStat.byteSent += (len(reqBuffer))
	servResp, _ := http.Post(servAddr, "application/json", bytes.NewReader(reqBuffer))
	body, _ := ioutil.ReadAll(servResp.Body)
	var respMsg Message
	json.Unmarshal(body, &respMsg)
	json.Unmarshal(respMsg.Data, &dst)
}
func claimHead(node *DNode, servAddr string) string {
	nodeBuffer, _ := json.Marshal(*node)
	reqMsg := Message{Data: nodeBuffer, Type: "claimHead"}
	reqBuffer, _ := json.Marshal(reqMsg)
	currentStat.serverRequest++
	currentStat.byteSent += (len(reqBuffer))
	servResp, _ := http.Post(servAddr, "application/json", bytes.NewReader(reqBuffer))

	body, _ := ioutil.ReadAll(servResp.Body)
	var respMsg Message
	json.Unmarshal(body, &respMsg)
	fileSig := respMsg.Sig
	return fileSig
}

func copyNode(dst *DNode, src *DNode) {
	dst.Attrs = src.Attrs
	for alias, nodes := range src.children {
		dst.children[alias] = nodes
	}
	for alias, sigs := range src.ChildSigs {
		dst.ChildSigs[alias] = sigs
	}
	dst.expanded = src.expanded
	dst.metaDirty = src.metaDirty
	dst.DataBlocks = make([]string, len(src.DataBlocks))
	copy(dst.DataBlocks, src.DataBlocks)
	dst.data = make([]uint8, len(src.data))
	copy(dst.data, src.data)
	// dst.dirty = src.dirty
}

func setDirty(node *DNode) {
	temp := node
	for temp != root {
		temp.metaDirty = true
		temp = temp.parent
	}
	temp.metaDirty = true
}

func GetLastHead(addr string) string {
	req := Message{Type: "lastclaim", Name: "HEAD"}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	currentStat.serverRequest++
	currentStat.byteSent += (len(jsonOutput))
	res, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	if err != nil {
		return ""
	}
	body, _ := ioutil.ReadAll(res.Body)
	var m1 = Message{}
	json.Unmarshal(body, &m1)
	return m1.Sig
}

func GetAllVersions(inode uint64) []string {
	var retVal []string
	req := Message{Type: "getAllVersions", Name: strconv.FormatUint(inode, 10)}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	currentStat.serverRequest++
	currentStat.byteSent += (len(jsonOutput))
	res, err := http.Post(servAddr, "application/json", bytes.NewReader(jsonOutput))
	if err != nil {
		fmt.Println("ERROR SEND POST")
		os.Exit(1)
	}
	body, _ := ioutil.ReadAll(res.Body)
	var m1 = Message{}
	json.Unmarshal(body, &m1)
	retVal = append(retVal, m1.Chain...)
	return retVal
}

func selectHead(timestamp string) string {
	req := Message{Type: "selectClaim", Info: timestamp}
	fmt.Println(timestamp)
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	currentStat.serverRequest++
	currentStat.byteSent += (len(jsonOutput))
	res, err := http.Post(servAddr, "application/json", bytes.NewReader(jsonOutput))
	if err != nil {
		PrintExit("NO SUCH HEAD\n")
	}
	if res.StatusCode != 200 {
		PrintExit("NO SUCH HEAD\n")
	}
	body, _ := ioutil.ReadAll(res.Body)
	var m1 = Message{}
	json.Unmarshal(body, &m1)
	return m1.Sig
}
