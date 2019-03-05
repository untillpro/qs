package utils

import "log"

// Doing prints obj followed by ...
func Doing(obj interface{}) {
	log.Print(obj, "...")
}
