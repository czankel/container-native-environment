package main

import (
	"fmt"
	"os"

	"github.com/czankel/cne/cli"
	_ "github.com/czankel/cne/runtime/containerd"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
