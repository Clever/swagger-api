package main

import (
	"log"
	"os"
	"strings"

	v1 "github.com/Clever/swagger-api/v1"
	v2 "github.com/Clever/swagger-api/v2"
	v3 "github.com/Clever/swagger-api/v3"
)

func main() {
	if len(os.Args) == 1 {
		log.Fatalf("You must supply a version to generate (VERSION=(1|2|3))")
	}

	version := strings.ToLower(os.Args[1])
	if version == "1" {
		v1.Generate()
	} else if version == "2" {
		v2.Generate()
	} else if version == "3" {
		v3.Generate()
	} else {
		log.Fatalf("Invalid version")
	}
}
