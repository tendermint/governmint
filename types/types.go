package types

import (
	"bytes"
	"fmt"
	"time"

	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-merkle"
	"github.com/tendermint/go-wire"
)

type Entity struct {
	ID     string        `json:"id"` // Unique
	PubKey crypto.PubKey `json:"pub_key"`
}

type Group struct {
	ID      string   `json:"id"` // Unique
	Version int      `json:"version"`
	Members []Member `json:"members"`
}

type Member struct {
	EntityID    string `json:"entity_id"`
	VotingPower int    `json:"voting_power"`
}

type Proposal struct {
	ID         string `json:"id"` // Unique
	Type       string `json:"type"`
	Data       string `json:"data"`
	GroupID    string `json:"group_id"`
	ProposerID string `json:"proposer_id"`
}

type Vote struct {
	Value bool `json:"value"`
}

type ActiveProposal struct {
	Proposal
	Votes []*Vote // same order as Group.Members

	votesFor     int
	votesAgainst int
}

//----------------------------------------

type ProposalTx struct {
	Proposal  `json:"proposal"`
	Signature crypto.Signature `json:"signature,omitempty"`
}

func (tx *ProposalTx) SignBytes() []byte {
	buf := new(bytes.Buffer)
	var err error
	var n int
	sig := tx.Signature
	tx.Signature = nil
	wire.WriteJSON(tx, buf, &n, &err)
	tx.Signature = sig
	return buf.Bytes()
}

type VoteTx struct {
	EntityID   string           `json:"entity_id"`
	ProposalID string           `json:"proposal_id"`
	Vote       Vote             `json:"vote"`
	Signature  crypto.Signature `json:"signature,omitempty"`
}

func (tx *VoteTx) SignBytes() []byte {
	buf := new(bytes.Buffer)
	var err error
	var n int
	sig := tx.Signature
	tx.Signature = nil
	wire.WriteJSON(tx, buf, &n, &err)
	tx.Signature = sig
	return buf.Bytes()
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
