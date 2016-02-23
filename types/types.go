package types

import (
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
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
	Value bool   `json:"value"`
	Text  string `json:"text"`
}

type ActiveProposal struct {
	Proposal
	Votes []*Vote // same order as Group.Members
}

//----------------------------------------

func EntityKey(entityID string) []byte {
	return []byte("G:e:" + entityID)
}

func GroupKey(groupID string) []byte {
	return []byte("G:g:" + groupID)
}

func ActiveProposalKey(proposalID string) []byte {
	return []byte("G:ap:" + proposalID)
}

//----------------------------------------

type Proposal interface {
	AssertIsProposal()
}

const (
	ProposalTypeGroupUpdate     = byte(0x01)
	ProposalTypeGroupCreate     = byte(0x02)
	ProposalTypeVariableSet     = byte(0x03)
	ProposalTypeText            = byte(0x04)
	ProposalTypeSoftwareUpgrade = byte(0x05)
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

type SoftwareUpgradeProposal struct {
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

func ProposalID(proposal Proposal) string {
	return Fmt("%X", wire.BinaryRipemd160(struct{ Proposal }{proposal}))
}

//----------------------------------------

type ProposalTx struct {
	EntityID  string           `json:"entity_id"`
	Proposal  Proposal         `json:"proposal"`
	Signature crypto.Signature `json:"signature,omitempty"`
}

func (tx *ProposalTx) SignBytes() []byte {
	sig := tx.Signature
	tx.Signature = nil
	jsonBytes := wire.JSONBytes(tx)
	tx.Signature = sig
	return jsonBytes
}

type VoteTx struct {
	EntityID   string           `json:"entity_id"`
	ProposalID string           `json:"proposal_id"`
	Vote       Vote             `json:"vote"`
	Signature  crypto.Signature `json:"signature,omitempty"`
}

func (tx *VoteTx) SignBytes() []byte {
	sig := tx.Signature
	tx.Signature = nil
	jsonBytes := wire.JSONBytes(tx)
	tx.Signature = sig
	return jsonBytes
}

type Tx interface {
	SignBytes() []byte
}

const (
	TxTypeProposal = byte(0x01)
	TxTypeVote     = byte(0x02)
)

var _ = wire.RegisterInterface(
	struct{ Tx }{},
	wire.ConcreteType{ProposalTx{}, TxTypeProposal},
	wire.ConcreteType{VoteTx{}, TxTypeVote},
)
