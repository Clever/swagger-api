package main

import (
	"log"
	"os"
	"strings"

	"github.com/Clever/swagger-api/V1"
	"github.com/Clever/swagger-api/V2"
)

func main() {
	if len(os.Args) == 1 {
		log.Fatalf("You must supply a version to generate: v1 or v2")
	}

	version := strings.ToLower(os.Args[1])
	if version == "v1" {
		V1.Generate()
	} else if version == "v2" {
		V2.Generate()
	} else {
		log.Fatalf("Invalid version")
	}
}
