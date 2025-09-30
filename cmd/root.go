package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "publify",
	Short: "Convert documents between formats for e-readers",
	Long: `Publify is a CLI tool for converting documents between formats,
optimized for e-readers like Kobo, Kindle, and others.

Currently supports:
- PDF to EPUB conversion with reader-specific optimizations
- Metadata editing for EPUB files
- EPUB extraction and compression for manual editing workflows`,
	Version: "0.1.0",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
}
