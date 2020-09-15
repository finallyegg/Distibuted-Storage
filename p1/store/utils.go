package store

import (
	"fmt"
	"os"
)

var Debug = false

func PrintDebug(s string, args ...interface{}) {
	if !Debug {
		return
	}
	fmt.Printf(s, args...)
}

func PrintAlways(s string, args ...interface{}) {
	fmt.Printf(s, args...)
}

func PrintExit(s string, args ...interface{}) {
	fmt.Printf(s, args...)
	os.Exit(1)
}
