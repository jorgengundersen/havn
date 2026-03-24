// havn manages the lifecycle of development environment containers.
package main

import (
	"os"

	"github.com/jorgengundersen/havn/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
