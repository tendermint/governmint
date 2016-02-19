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

type Proposal interface {
	AssertIsProposal()
}

const (
	ProposalTypeGroupUpdate            = byte(0x01)
	ProposalTypeGroupCreate            = byte(0x02)
	ProposalTypeVariableSet            = byte(0x03)
	ProposalTypeTextProposal           = byte(0x04)
	ProposalTypeSoftwareUpdateProposal = byte(0x05)
)

type GroupUpdateProposal struct {
	GroupID    string   `json:"group_id"`
	AddMembers []Member `json:"add_members"`
	RemMembers []Member `json:"rem_members"`
}

type GroupCreateProposal struct {
	GroupID string   `json:"group_id"`
	Members []Member `json:"members"`
}

type VariableSetProposal struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TextProposal struct {
	Text string `json:"text"`
}

type SoftwareUpdateProposal struct {
	Module string `json:"module"`
	URL    string `json:"url"`
	Hash   []byte `json:"hash"`
}

func (_ GroupUpdateProposal) AssertIsProposal()     {}
func (_ GroupCreateProposal) AssertIsProposal()     {}
func (_ VariableSetProposal) AssertIsProposal()     {}
func (_ TextProposal) AssertIsProposal()            {}
func (_ SoftwareUpgradeProposal) AssertIsProposal() {}

var _ = wire.RegisterInterface(
	struct{ Proposal }{},
	wire.ConcreteType{GroupUpdateProposal{}, ProposalTypeGroupUpdate},
	wire.ConcreteType{GroupCreateProposal{}, ProposalTypeGroupCreate},
	wire.ConcreteType{VariableSetProposal{}, ProposalTypeVariableSet},
	wire.ConcreteType{TextProposal{}, ProposalTypeText},
	wire.ConcreteType{SoftwareUpgradeProposal{}, ProposalTypeSoftwareUpgrade},
)

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

type Tx interface{}

const (
	TxTypeProposal = byte(0x01)
	TxTypeVote     = byte(0x02)
)

var _ = wire.RegisterInterface(
	struct{ Tx }{},
	wire.ConcreteType{ProposalTx{}, txTypeProposal},
	wire.ConcreteType{VoteTx{}, txTypeVote},
)
