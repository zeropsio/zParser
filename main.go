package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	inName := "zerops-import.yml"
	outName := "zerops-import.parsed.yml"

	yml, err := os.Open(inName)
	if err != nil {
		// create output file in the same folder as input file
		inName = "../" + inName
		outName = "../" + outName

		yml, err = os.Open(inName)
		if err != nil {
			log.Fatal(err)
		}
	}

	out, err := os.OpenFile(outName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	p := NewImportParser(yml, out)

	if err := p.Parse(); err != nil {
		log.Fatal(err)
	}

	result, err := os.ReadFile(outName)
	fmt.Println(string(result))
}
