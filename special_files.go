package main

import (
	"strings"
)

var SpecialFileNamePrefixes []string = []string{"/run", "/dev"}

var Num int = len(SpecialFileNamePrefixes)

func specialFile(filename string) bool {
	for i := 0; i < Num; i++ {
		if strings.HasPrefix(filename, SpecialFileNamePrefixes[i]) {
			return true
		}
	}
	return false
}
