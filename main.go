package main

import (
	"FileCleanup/cmd"
	_ "FileCleanup/cmd/performanceBenchmark"
	_ "FileCleanup/pkg"
)

func main() {
	cmd.Execute()
}
