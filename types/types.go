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

func NewMember(entityID string, votingPower int) *Member {
	return &Member{entityID, votingPower}
}

type Vote struct {
	Value     string           `json:"value"`
	Signature crypto.Signature `json:"signature"`
}

func NewVote(value string, sig crypto.Signature) *Vote {
	return &Vote{value, sig}
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
	ProposalTypeGroupCreate     = byte(0x01)
	ProposalTypeGroupUpdate     = byte(0x02)
	ProposalTypeText            = byte(0x11)
	ProposalTypeSoftwareUpgrade = byte(0x12)
)

type GroupCreateProposal struct {
	GroupID string   `json:"group_id"`
	Members []Member `json:"members"`
}

type GroupUpdateProposal struct {
	GroupID      string   `json:"group_id"`
	GroupVersion int      `json:"group_version"`
	AddMembers   []Member `json:"add_members"`
	RemMembers   []Member `json:"rem_members"`
}

type TextProposal struct {
	Text string `json:"text"`
}

type SoftwareUpgradeProposal struct {
	Module string `json:"module"`
	URL    string `json:"url"`
	Hash   []byte `json:"hash"`
}

func (_ *GroupCreateProposal) AssertIsProposal()     {}
func (_ *GroupUpdateProposal) AssertIsProposal()     {}
func (_ *TextProposal) AssertIsProposal()            {}
func (_ *SoftwareUpgradeProposal) AssertIsProposal() {}

var _ = wire.RegisterInterface(
	struct{ Proposal }{},
	wire.ConcreteType{&GroupCreateProposal{}, ProposalTypeGroupCreate},
	wire.ConcreteType{&GroupUpdateProposal{}, ProposalTypeGroupUpdate},
	wire.ConcreteType{&TextProposal{}, ProposalTypeText},
	wire.ConcreteType{&SoftwareUpgradeProposal{}, ProposalTypeSoftwareUpgrade},
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
	Value      string           `json:"value"`
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
