package main

import (
	"bytes"
	"encoding/hex"
	"os"
	"strconv"

	"github.com/codegangsta/cli"
	. "github.com/tendermint/go-common"
	rpcclient "github.com/tendermint/go-rpc/client"
	"github.com/tendermint/go-wire"
	gov "github.com/tendermint/governmint"
	rpctypes "github.com/tendermint/tendermint/rpc/core/types"
)

var (
	TendermintHost = "http://localhost:46657"

	groupFlag = cli.StringFlag{
		Name:  "group",
		Value: "Guvnahs",
		Usage: "Group name",
	}

	proposerFlag = cli.StringFlag{
		Name:  "proposer",
		Value: "entity1",
		Usage: "Name of proposer",
	}

	keyFlag = cli.StringFlag{
		Name:  "key",
		Value: "key1",
		Usage: "Key to use for signing (by name)",
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
			Usage:     "Broadcast a proposal",
			ArgsUsage: "[proposal name] [data]",
			Action: func(c *cli.Context) {
				cmdPropose(c)
			},
			Flags: []cli.Flag{
				groupFlag,
				proposerFlag,
				keyFlag,
			},
		},
		{
			Name:      "vote",
			Usage:     "Vote on a proposal",
			ArgsUsage: "[proposal name] [member index] [vote]",
			Action: func(c *cli.Context) {
				cmdVote(c)
			},
			Flags: []cli.Flag{
				keyFlag,
			},
		},
	}
	// app.Before = before
	app.Run(os.Args)
}

/* clients need

- a members index in the group for vote txs
- proposal ids ...

*/

func cmdPropose(c *cli.Context) {
	args := c.Args()
	if len(args) != 2 {
		Exit("propose takes two args")
	}

	name, data := args[0], args[1]

	pTx := gov.ProposalTx{
		Name:       name,
		Data:       data,
		GroupID:    c.String("group"),
		ProposerID: c.String("proposer"),
	}

	sig, err := SignTx(pTx, c.String("key"))
	if err != nil {
		Exit(err.Error())
	}
	tx := &gov.SignedTx{pTx, sig}
	buf := new(bytes.Buffer)
	var n int
	wire.WriteJSON(tx, buf, &n, &err)
	if err != nil {
		Exit(err.Error())
	}
	client := rpcclient.NewClientJSONRPC(TendermintHost)
	_, err = client.Call(
		"broadcast_tx",
		[]interface{}{
			hex.EncodeToString(buf.Bytes())},
		&rpctypes.ResultBroadcastTx{},
	)
	if err != nil {
		Exit(err.Error())
	}
	log.Notice("Broadcast proposal", "id", []byte((&gov.Proposal{ProposalTx: &pTx}).ID()), "sig", sig)
}

func cmdVote(c *cli.Context) {
	args := c.Args()
	if len(args) != 3 {
		Exit("propose takes three args")
	}

	proposal, memberS, voteS := args[0], args[1], args[2]

	member, err := strconv.Atoi(memberS)
	if err != nil {
		Exit(err.Error())
	}

	var vote bool
	switch voteS {
	case "yes", "true", "1", "for":
		vote = true
	case "no", "false", "0", "against":
		vote = false
	default:
		Exit("Unknown vote " + voteS)

	}

	vTx := gov.VoteTx{
		ProposalID: proposal,
		Member:     member,
		Vote:       vote,
	}

	sig, err := SignTx(vTx, c.String("key"))
	if err != nil {
		Exit(err.Error())
	}
	tx := &gov.SignedTx{vTx, sig}
	buf := new(bytes.Buffer)
	var n int
	wire.WriteJSON(tx, buf, &n, &err)
	if err != nil {
		Exit(err.Error())
	}
	client := rpcclient.NewClientJSONRPC(TendermintHost)
	_, err = client.Call(
		"broadcast_tx",
		[]interface{}{hex.EncodeToString(buf.Bytes())},
		&rpctypes.ResultBroadcastTx{},
	)
	if err != nil {
		Exit(err.Error())
	}
	log.Notice("Broadcast vote", "tx", vTx, "sig", sig)
}
