package main

import (
	"os"

	"github.com/hichuyamichu/myriag/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
