package main

import (
	"fmt"
	"os"

	"github.com/ibc-scouts/ibc-interceptor/cmd"
)

func main() {
	rootCmd := cmd.RootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
