package main

import (
	"fmt"

	//rst "github.com/siongui/go-rst"
	".."
)

func main() {
	lines := rst.File2lines("README.rst")

	for _, line := range lines {
		fmt.Println(line)
	}
}
