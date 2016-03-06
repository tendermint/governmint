package types

import (
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
)

const (
	AdminGroupID = "admin"
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
	VotingPower uint64 `json:"voting_power"`
}

func NewMember(entityID string, votingPower int) *Member {
	return &Member{entityID, votingPower}
}

type SignedVote struct {
	Value     string           `json:"value"`
	Signature crypto.Signature `json:"signature"`
}

func NewSignedVote(value string, sig crypto.Signature) *SignedVote {
	return &SignedVote{value, sig}
}

type ActiveProposal struct {
	Proposal
	SignedVotes []*SignedVote // same order as Group.Members
}

type Vote struct {
	ProposalID string `json:"proposal_id"`
	Value      string `json:"value"`
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

type GroupCreateProposal struct {
	GroupID string   `json:"group_id"`
	Members []Member `json:"members"`
}

func (p GroupCreateProposal) GetGroupID() string { return p.GroupID }

type GroupUpdateProposal struct {
	GroupID        string   `json:"group_id"`
	GroupVersion   int      `json:"group_version"`
	ChangedMembers []Member `json:"changed_members"` // 0 VotingPower to remove
}

func (p GroupUpdateProposal) GetGroupID() string { return p.GroupID }

type TextProposal struct {
	GroupID string `json:"group_id"`
	Text    string `json:"text"`
}

func (p TextProposal) GetGroupID() string { return p.GroupID }

type UpgradeProposalModule struct {
	Name   string `json:"module"`
	Script string `json:"script"`
}

type UpgradeProposal struct {
	Modules []UpgradeProposalModule
}

func (p UpgradeProposal) GetGroupID() string { return AdminGroupID }

func ProposalID(proposal Proposal) string {
	return Fmt("%X", wire.BinaryRipemd160(struct{ Proposal }{proposal}))
}

type ProposalWithID struct {
	Proposal Proposal `json:"unwrap"`
	id       string   `json:"-"`
}

func (p *ProposalWithID) ID() string {
	if p.id == "" {
		p.id = ProposalID(p.Proposal)
	}
	return p.id
}

type Proposal interface {
	AssertIsProposal()
	GetGroupID() string
}

const (
	ProposalTypeGroupCreate = byte(0x01)
	ProposalTypeGroupUpdate = byte(0x02)
	ProposalTypeText        = byte(0x11)
	ProposalTypeUpgrade     = byte(0x12)
)

func (_ *GroupCreateProposal) AssertIsProposal() {}
func (_ *GroupUpdateProposal) AssertIsProposal() {}
func (_ *TextProposal) AssertIsProposal()        {}
func (_ *UpgradeProposal) AssertIsProposal()     {}

var _ = wire.RegisterInterface(
	struct{ Proposal }{},
	wire.ConcreteType{&GroupCreateProposal{}, ProposalTypeGroupCreate},
	wire.ConcreteType{&GroupUpdateProposal{}, ProposalTypeGroupUpdate},
	wire.ConcreteType{&TextProposal{}, ProposalTypeText},
	wire.ConcreteType{&UpgradeProposal{}, ProposalTypeUpgrade},
)

//----------------------------------------

// A simple tx to be signed by a single entity
type SimpleTx interface {
	EntityID() string
	Signature() crypto.Signature
}

type ProposalTx struct {
	EntityID  string           `json:"entity_id"`
	Proposal  ProposalWithID   `json:"proposal"`
	Signature crypto.Signature `json:"signature"`
}

func (tx *ProposalTx) EntityID() string            { return tx.EntityID }
func (tx *ProposalTx) Signature() crypto.Signature { return tx.Signature }
func (tx *ProposalTx) SignBytes() []byte {
	return wire.JSONBytes(tx.Proposal)
}

type VoteTx struct {
	EntityID  string           `json:"entity_id"`
	Vote      Vote             `json:"vote"`
	Signature crypto.Signature `json:"signature"`
}

func (tx *VoteTx) EntityID() string            { return tx.EntityID }
func (tx *VoteTx) Signature() crypto.Signature { return tx.Signature }
func (tx *VoteTx) SignBytes() []byte {
	return wire.JSONBytes(tx.Vote)
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

//----------------------------------------

type TMSPError struct {
	Code tmsp.CodeType
	Log  string
}

func (tmspErr TMSPError) IsOK() bool {
	return tmspErr.Code == tmsp.CodeType_OK
}
