// Package main is the entry point for the faas CLI.
package main

import "os"

var (
	version = "dev"
	commit  = "none"
)

func main() {
	if err := Execute(version, commit); err != nil {
		os.Exit(1)
	}
}
