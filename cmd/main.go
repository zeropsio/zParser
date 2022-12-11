package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	_ "time/tzdata"

	"git.vsh-labs.cz/zerops/zparser/src/metaError"
	"git.vsh-labs.cz/zerops/zparser/src/parser"

	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:   "parse",
		Short: "parses provided file",
		Args:  cobra.ExactArgs(1),
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			f, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("failed to open file [%s]: %w", args[0], err)
			}

			var out io.Writer
			outFile, err := cmd.Flags().GetString("output-file")
			if err != nil {
				return fmt.Errorf("failed to read output-file flag: %w", err)
			}
			if outFile != "" {
				var err error
				out, err = os.OpenFile(outFile, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
				if err != nil {
					return fmt.Errorf("failed to open output file [%s]: %w", outFile, err)
				}
			} else {
				out = os.Stdout
			}

			maxFunctions, err := cmd.Flags().GetInt("max-functions")
			if err != nil {
				return fmt.Errorf("failed to read max-functions flag: %w", err)
			}

			p := parser.NewParser(f, out, maxFunctions)
			return p.Parse(cmd.Context())
		},
	}

	cmd.Flags().StringP("output-file", "f", "", "path to the file where result will be saved to, if not set, stdOut is used")
	cmd.Flags().Int("max-functions", 200, "max amount of function calls that may occur during parsing of the provided file")

	if err := cmd.Execute(); err != nil {
		metaErr := new(metaError.MetaError)
		if errors.As(err, &metaErr) {
			metaErr.Print()
			os.Exit(1)
		}
		log.Fatal(err)
	}
}
