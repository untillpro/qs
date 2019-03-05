package utils

import (
	"fmt"
	"os"
)

// https://stackoverflow.com/questions/25190971/golang-copy-exec-output-to-log

// PanicIfError panics if err is not null
func PanicIfError(err error) {
	if nil != err {
		panic(err)
	}
}

// ExitIfError if err is not null
func ExitIfError(err error, args ...interface{}) {
	if nil != err {
		fmt.Fprintln(os.Stderr, args...)
		os.Exit(1)
	}
}

// Assert exists if cond is not true
func Assert(cond bool, args ...interface{}) {
	if !cond {
		fmt.Fprintln(os.Stderr, args...)
		os.Exit(1)
	}
}
