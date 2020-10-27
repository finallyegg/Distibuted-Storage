package store

import (
	"fmt"
	"io/ioutil"
	"os"
)

var (
	Debug = true
)

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
	os.Exit(1)
}

func PrintAlways(s string, args ...interface{}) {
	fmt.Printf(s, args...)
}

func updateLast(sig string) {
	if err := ioutil.WriteFile("./last", []byte(sig), 0777); err != nil {
		PrintAlways("\nERROR: Unable to write %q to %q\n", sig, ".last")
	}
}

func getLast() string {
	bytes, err := ioutil.ReadFile("./last")
	if err != nil {
		return "none"
	} else {
		return string(bytes)
	}
}
