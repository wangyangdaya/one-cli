package main

import (
	"os"

	"one-cli/internal/app"
)

func main() {
	os.Exit(app.ExecuteRoot(app.NewRootCommand()))
}
