package main

import (
	".."
	"fmt"
)

func main() {
	lines := rst.File2lines("README.rst")

	for _, line := range lines {
		fmt.Println(line)
	}
}
