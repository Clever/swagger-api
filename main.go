package main

import (
	"log"
	"os"
	"strings"

	"github.com/Clever/swagger-api/v1"
	"github.com/Clever/swagger-api/v2"
)

func main() {
	if len(os.Args) == 1 {
		log.Fatalf("You must supply a version to generate: v1 or v2")
	}

	version := strings.ToLower(os.Args[1])
	if version == "v1" {
		v1.Generate()
	} else if version == "v2" {
		v2.Generate()
	} else {
		log.Fatalf("Invalid version")
	}
}
