package main

import (
	"os"

	. "gitlab.cs.umd.edu/cmsc818eFall20/cmsc818e-zepinghe/p4/store"
)

//=====================================================================

func main() {

	_, args := os.Args[0], os.Args[1:]

	if len(args) > 0 && args[0] == "-d" {
		args = args[1:]
		// Debug = !Debug
	}

	if len(args) < 1 {
		PrintExit("Server argument error'\n")
	}

	var sqlPath string
	if len(args) < 2 {
		sqlPath = "./db_default"
	} else {
		sqlPath = args[1]
	}
	serverAddress := "localhost:" + args[0]
	Serve(serverAddress, sqlPath)
}
