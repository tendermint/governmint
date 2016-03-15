package main

import (
	"os"

	"github.com/codegangsta/cli"
	. "github.com/tendermint/go-common"
	types "github.com/tendermint/governmint/types"
)

var (
	groupFlag = cli.StringFlag{
		Name:  "group",
		Value: types.ValidatorsGroupID,
		Usage: "Group ID",
	}
)

func main() {
	app := cli.NewApp()
	app.Name = "gov-cli"
	app.Usage = "gov-cli [command] [args...]"
	app.Flags = []cli.Flag{}
	app.Commands = []cli.Command{
		{
			Name:      "propose",
			Usage:     "Create and sign a proposal",
			ArgsUsage: "[proposal type]",
			Action: func(c *cli.Context) {
				cmdPropose(c)
			},
			Flags: []cli.Flag{
				groupFlag,
			},
		},
	}
	// app.Before = before
	app.Run(os.Args)
}

func cmdPropose(c *cli.Context) {
	args := c.Args()
	if len(args) != 2 {
		Exit("propose takes two args")
	}

	log.Info("args", args)
}
