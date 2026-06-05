package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Jubblin/ip-anonymizer/internal/anonymize"
)

func main() {
	input := flag.String("input", "", "Source JSON file (required)")
	output := flag.String("output", "", "Destination JSON file (default: stdout)")
	mapping := flag.String("mapping", ".ip-anonymizer-mapping.json", "Secret mapping file path")
	inPlace := flag.Bool("in-place", false, "Overwrite the input file")
	flag.Parse()

	if *input == "" {
		fmt.Fprintln(os.Stderr, "error: -input is required")
		flag.Usage()
		os.Exit(1)
	}

	outputPath := *output
	if *inPlace {
		if outputPath != "" {
			fmt.Fprintln(os.Stderr, "error: -output cannot be used with -in-place")
			os.Exit(1)
		}
		outputPath = *input + ".tmp"
	}

	if err := anonymize.ProcessFile(*input, outputPath, *mapping); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *inPlace {
		if err := os.Rename(outputPath, *input); err != nil {
			_ = os.Remove(outputPath)
			fmt.Fprintf(os.Stderr, "error: replace input file: %v\n", err)
			os.Exit(1)
		}
	}
}
