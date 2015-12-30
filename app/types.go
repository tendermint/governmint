package governmint

import (
	"time"

	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
)

type Entity struct {
	Name      string
	PubKey    crypto.PubKey
	InvitedBy string
	Invites   int
}

type Member struct {
	*Entity
	VotingPower int
}

type Group struct {
	Name        string // XXX Unique constraints?
	Version     int
	LastUpdated time.Time
	Members     []*Member
}

type Proposal struct {
	Data string
	*Group
	Votes []crypto.Signature // same order as Group.Members

	passed bool
}

//-----------------------------

type Tx interface {
	// TODO
	// WriteSignBytes(chainID string, w io.Writer, n *int64, err *error)
}

const (
	txTypeProposal = byte(0x01)
	txTypeVote     = byte(0x02)
)

var _ = wire.RegisterInterface(
	struct{ Tx }{},
	wire.ConcreteType{ProposalTx{}, txTypeProposal},
	wire.ConcreteType{VoteTx{}, txTypeVote},
)

type ProposalTx struct {
	Data     string `json:"data"`
	Group    string `json:"group"`
	Proposer string `json:"proposer"`

	Signature crypto.Signature `json:"signature"`
}

type VoteTx struct {
	Proposal string `json:"proposal"`
	Vote     bool   `json:"vote"`
	Member   string `json:"member"`

	Signature crypto.Signature `json:"signature"`
}
