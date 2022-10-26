package main

import (
	"fmt"
	"os"
)

func main() {
	yml, err := os.Open("zerops-import.yml")
	if err != nil {
		panic(err)
	}

	// TODO(ms): add support for writeString method :-(
	p := NewImportParser(yml)

	if err := p.Parse(); err != nil {
		panic(err)
	}

	fmt.Println(p.GetOutput())
}
