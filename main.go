package main

import (
	. "github.com/tendermint/go-common"
	app "github.com/tendermint/governmint/app"
	"github.com/tendermint/tmsp/server"
)

func main() {

	// Start the listener
	_, err := server.StartListener("tcp://0.0.0.0:46658", app.NewGovernmintApplication())
	if err != nil {
		Exit(err.Error())
	}

	// Wait forever
	TrapSignal(func() {
		// Cleanup
	})

}
