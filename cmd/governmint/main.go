package main

import (
	"os"

	. "github.com/tendermint/go-common"
	gov "github.com/tendermint/governmint"
	"github.com/tendermint/tmsp/server"
)

func main() {

	var govFile string
	if len(os.Args) == 1 {
		Exit("Please specify the governmint genesis file")
	}

	govFile = os.Args[1]

	// Start the listener
	_, err := server.StartListener("tcp://0.0.0.0:46658", gov.NewGovernmintApplication(govFile))
	if err != nil {
		Exit(err.Error())
	}

	// Wait forever
	TrapSignal(func() {
		// Cleanup
	})

}
