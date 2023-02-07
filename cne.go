package main

import (
	"fmt"
	"os"

	"github.com/czankel/cne/cli"
	_ "github.com/czankel/cne/runtime/containerd"
	_ "github.com/czankel/cne/runtime/remote"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
