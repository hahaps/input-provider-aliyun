package main

import (
	"github.com/hahaps/common-provider/src/input"
	"github.com/hahaps/input-provider-aliyun/src"
	"log"
	"strings"
)

var VERSION string = "v0.1"

type Version struct {}

func (Version)Check(version string, matched *bool) error {
	*matched = strings.Compare(VERSION, version) == 0
	return nil
}

func main() {
	if err := input.RunProvider(Version{}, src.ResourceMap); err != nil {
		log.Fatal(err)
	}
}
