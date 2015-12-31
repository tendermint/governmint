package main

import (
	. "github.com/tendermint/go-common"
	gov "github.com/tendermint/governmint"
	"github.com/tendermint/tmsp/server"
)

func main() {

	// Start the listener
	_, err := server.StartListener("tcp://0.0.0.0:46658", gov.NewGovernmintApplication())
	if err != nil {
		Exit(err.Error())
	}

	// Wait forever
	TrapSignal(func() {
		// Cleanup
	})

}
