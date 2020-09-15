package main

import (
	//. "gitlab.cs.umd.edu/cmsc818eFall20/cmsc818e-keleher/p1/store"
	. "./store"
	"os"
)

//=====================================================================

func main() {

	if len(os.Args) < 3 {
		usage()
	}

	//p_out("nice\n")
	_, args := os.Args[0], os.Args[1:]

	if args[0] == "-d" {
		args = args[1:]
		Debug = !Debug
	}
	cmd := args[0]
	arg := args[1]
	args = args[2:]
	//p_out("ARG %q\n", arg)

	switch cmd {

	case "put":
		CmdPut(arg)

	case "get":
		if len(args) > 0 {
			CmdGet(arg, args[0])
		} else {
			usage()
		}

	case "desc":
		CmdDesc(arg)

	default:
		usage()
	}
}

func usage() {
	PrintExit("USAGE: (put <path> | get <sig> <new path> | desc <sig>)\n")
}
