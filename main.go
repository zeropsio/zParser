package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	inName := "zerops-import.yml"
	outName := "zerops-import.parsed.yml"

	// TODO(ms): GRPC api with context used to stop execution and option to customize max function call count

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
	p := NewImportParser(yml, out, 200)

	s := time.Now()
	if err := p.Parse(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("parser took: %s\n\n\n", time.Since(s).String())

	result, err := os.ReadFile(outName)
	fmt.Println(string(result))
}
