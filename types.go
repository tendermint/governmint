package governmint

import (
	"bytes"
	"time"

	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-merkle"
	"github.com/tendermint/go-wire"
)

type Entity struct {
	Name      string
	PubKey    crypto.PubKey
	InvitedBy string
	Invites   int
}

type Member struct {
	Entity      []byte
	VotingPower int
}

type Group struct {
	Name        string // XXX Unique constraints?
	Version     int
	LastUpdated time.Time
	Members     []*Member
}

type Proposal struct {
	Data  string
	Group []byte
	Votes []Vote // same order as Group.Members

	votesFor     int
	votesAgainst int
}

type Vote struct {
	Vote      bool
	Signature crypto.Signature
}

//-----------------------------

type SignedTx struct {
	Tx        Tx               `json:"tx"`
	Signature crypto.Signature `json:"signature"`
}

type Tx interface {
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

func Hash(o interface{}) []byte {
	buf := new(bytes.Buffer)
	var err error
	var n int
	wire.WriteBinary(o, buf, &n, &err)
	return merkle.SimpleHashFromBinary(buf.Bytes())
}

func SignBytes(tx Tx) []byte {
	buf := new(bytes.Buffer)
	var err error
	var n int
	wire.WriteBinary(tx, buf, &n, &err)
	return buf.Bytes()
}

func TxID(tx Tx) []byte {
	return merkle.SimpleHashFromBinary(SignBytes(tx))
}

type ProposalTx struct {
	Data     string `json:"data"`
	Group    []byte `json:"group"`
	Proposer []byte `json:"proposer"`
}

type VoteTx struct {
	Proposal []byte `json:"proposal"`
	Vote     bool   `json:"vote"`
	Member   int    `json:"member"`
}
