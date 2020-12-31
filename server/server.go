package main

import (
	"os"

	. "github.com/mattn/go-getopt"
	. "gitlab.cs.umd.edu/cmsc818eFall20/cmsc818e-zepinghe/p5b/store"
)

//=====================================================================

func main() {

	var c int
	Debug := false
	servAddr := ""
	publicKeyPath := ""
	anchorSigned := false
	sqlPath := "./db_default"
	for {
		if c = Getopt("dp:xs:b:"); c == EOF {
			break
		}

		switch c {
		case 'd':
			// fmt.Println("d", c)
			Debug = !Debug
		case 'p':
			// fmt.Println("p", c)
			publicKeyPath = OptArg
		case 'x':
			// fmt.Println("x", c)
			anchorSigned = !anchorSigned
		case 's':
			// fmt.Println("s", c)
			servAddr = "localhost:" + OptArg
		case 'b':
			// fmt.Println("b", c)
			sqlPath = OptArg
		default:

			println("usage: main.go [-d | -p <public key> | -x <do achor need signed?>]", c)
			os.Exit(1)
		}
	}
	Serve(servAddr, sqlPath, publicKeyPath, anchorSigned)
}
