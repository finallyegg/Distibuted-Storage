package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	. "gitlab.cs.umd.edu/cmsc818eFall20/cmsc818e-zepinghe/p4/store"
)

func main() {

	if len(os.Args) < 3 {
		usage()
	}

	_, args := os.Args[0], os.Args[1:]

	if len(args[0]) > 0 && args[0] == "-d" {
		args = args[1:]
		Debug = !Debug
	}

	addrPtr := flag.String("s", "localhost: 8000", "servAddr")
	priKeyPtr := flag.String("q", "", "Somepath")
	pubKeyPtr := flag.String("p", "", "somepath")
	flag.Parse()

	flagCount := 0
	if *addrPtr != "" {
		flagCount += 2

	}
	if *priKeyPtr != "" {
		flagCount += 2

	}
	if *pubKeyPtr != "" {
		flagCount += 2
	}
	// fmt.Println("addr:", *addrPtr)
	// fmt.Println("pri:", *priKeyPtr)
	// fmt.Println("pub:", *pubKeyPtr)

	args = args[flagCount:]

	cmd, args := args[0], args[1:]
	addr_1 := "http://" + *addrPtr + "/json"

	switch cmd {
	case "genkeys":
		var publicKeyBlock pem.Block
		privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		publicKeyBlock.Type = "RSA PUBLIC KEY"
		publicKeyBlock.Bytes = x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)

		var privateKeyBlock pem.Block
		privateKeyBlock.Type = "RSA PRIVATE KEY"
		privateKeyBlock.Bytes = x509.MarshalPKCS1PrivateKey(privateKey)

		publicBody := pem.EncodeToMemory(&publicKeyBlock)
		ioutil.WriteFile("key.public", publicBody, 0777)

		privateBody := pem.EncodeToMemory(&privateKeyBlock)
		ioutil.WriteFile("key.private", privateBody, 0777)

	case "get":
		PrintAssert(len(args) >= 2, "USAGE: get <sig> <path>\n")
		CmdGetFile(addr_1, args[0], args[1], "./bloblocal")

	case "getfile":
		PrintAssert(len(args) >= 2, "USAGE: get <sig> <path>\n")
		CmdGetFile(addr_1, args[0], args[1], "./bloblocal")

	case "getsig":
		PrintAssert(len(args) >= 2, "USAGE: get <sig> <path>\n")
		addr_1 := "http://" + *addrPtr
		CmdGetFileNoJSON(addr_1, args[0], args[1])

	case "put":
		PrintAssert(len(args) >= 1, "USAGE: put <sig>\n")
		CmdPut(addr_1, args[0])

	case "desc":
		PrintAssert(len(args) >= 1, "USAGE: desc <sig>\n")
		ret := CmdDesc(addr_1, args[0])
		fmt.Println(ret)

	case "del":
		PrintAssert(len(args) >= 1, "USAGE: del <sig>\n")
		CmdDel(addr_1, args[0])

	case "info":
		CmdInfo(addr_1)

	case "sign":
		CmdSign(addr_1, args[0], *priKeyPtr)

	case "verify":
		CmdVerify(addr_1, args[0], *pubKeyPtr)
	case "anchor":
		CmdCreateAnchor(addr_1, args[0])

	case "claim":
		CmdClaimAnchor(addr_1, args[0], args[1], args[2])

	case "content":
		CmdContent(addr_1, args[0])

	case "rootanchor":
		CmdRootanchor(addr_1, args[0])

	case "lastclaim":
		CmdLastClaim(addr_1, args[0])

	case "chain":
		CmdChain(addr_1, args[0])
	// case "sync":
	// 	PrintAssert(len(args) >= 2, "sync <addr1> <addr2> <height>\n")
	// 	i, _ := strconv.Atoi(args[1])
	// 	PrintAssert(i >= 1, "height >=1\n")
	// 	addr_2 := "http://" + args[0] + "/json"
	// 	CmdSync(addr_1, addr_2, i)
	default:
		usage()
	}
}

func usage() {
	PrintExit("USAGE: client <serveraddr> (put <path> | putfile <path> | getsig <sig> <new path> | desc <sig>)\n")
}
