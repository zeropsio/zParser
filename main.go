package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	yml, err := os.Open("zerops-import.yml")
	if err != nil {
		yml, err = os.Open("../zerops-import.yml")
		if err != nil {
			panic(err)
		}
	}

	p := NewImportParser(yml)

	if err := p.Parse(); err != nil {
		log.Fatal(err)
	}

	fmt.Println(p.GetOutput())
}
