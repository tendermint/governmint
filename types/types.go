package types

import (
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
)

const (
	AdminGroupID      = "admin"
	ValidatorsGroupID = "validators"
)

type Entity struct {
	Addr   []byte        `json:"addr"`
	PubKey crypto.PubKey `json:"pub_key"`
}

type Group struct {
	ID       string   `json:"id"`
	ParentID string   `json:"parent_id"`
	Version  int      `json:"version"`
	Members  []Member `json:"members"`
}

type Member struct {
	EntityAddr  []byte `json:"entity_addr"`
	VotingPower uint64 `json:"voting_power"`
}

func NewMember(entityAddr []byte, votingPower uint64) Member {
	return Member{entityAddr, votingPower}
}

type Vote struct {
	Height     uint64 `json:"height"`
	EntityAddr []byte `json:"entity_addr"`
	ProposalID string `json:"proposal_id"`
	Value      string `json:"value"`
}

// XXX What about chainID?
func (vote Vote) SignBytes() []byte { return wire.JSONBytes(vote) }

type SignedVote struct {
	Vote      Vote             `json:"vote"`
	Signature crypto.Signature `json:"signature"`
}

func NewSignedVote(vote Vote, sig crypto.Signature) SignedVote {
	return SignedVote{vote, sig}
}

type Proposal struct {
	ID          string       `json:"id"`
	VoteGroupID string       `json:"vote_group_id"`
	StartHeight uint64       `json:"start_height"`
	EndHeight   uint64       `json:"end_height"`
	Info        ProposalInfo `json:"info"`
}

func (proposal Proposal) SignBytes() []byte { return wire.JSONBytes(proposal) }

type ActiveProposal struct {
	Proposal    `json:"proposal"`
	SignedVotes []SignedVote `json:"signed_votes"`
}

//----------------------------------------

type GroupCreateProposalInfo struct {
	NewGroupID string   `json:"new_group_id"` // The new group's ID
	Members    []Member `json:"members"`      // The members of the new group
}

type GroupUpdateProposalInfo struct {
	UpdateGroupID  string   `json:"update_group_id"` // The group to update
	NextVersion    int      `json:"next_version"`    // The group's version, bumped 1
	ChangedMembers []Member `json:"changed_members"` // 0 VotingPower to remove
}

type TextProposalInfo struct {
	Text string `json:"text"`
}

type UpgradeProposalInfoModule struct {
	Name   string `json:"module"`
	Script string `json:"script"`
}

type UpgradeProposalInfo struct {
	Modules []UpgradeProposalInfoModule
}

type ProposalInfo interface {
	AssertIsProposalInfo()
}

const (
	ProposalInfoTypeGroupCreate = byte(0x01)
	ProposalInfoTypeGroupUpdate = byte(0x02)
	ProposalInfoTypeText        = byte(0x11)
	ProposalInfoTypeUpgrade     = byte(0x12)
)

func (_ *GroupCreateProposalInfo) AssertIsProposalInfo() {}
func (_ *GroupUpdateProposalInfo) AssertIsProposalInfo() {}
func (_ *TextProposalInfo) AssertIsProposalInfo()        {}
func (_ *UpgradeProposalInfo) AssertIsProposalInfo()     {}

var _ = wire.RegisterInterface(
	struct{ ProposalInfo }{},
	wire.ConcreteType{&GroupCreateProposalInfo{}, ProposalInfoTypeGroupCreate},
	wire.ConcreteType{&GroupUpdateProposalInfo{}, ProposalInfoTypeGroupUpdate},
	wire.ConcreteType{&TextProposalInfo{}, ProposalInfoTypeText},
	wire.ConcreteType{&UpgradeProposalInfo{}, ProposalInfoTypeUpgrade},
)

//----------------------------------------

type ProposalTx struct {
	EntityAddr []byte           `json:"entity_addr"`
	Proposal   Proposal         `json:"proposal"`
	Signature  crypto.Signature `json:"signature"`
}

func (tx *ProposalTx) SignBytes() []byte { return tx.Proposal.SignBytes() }
func (tx *ProposalTx) SetSignature(pub crypto.PubKey, sig crypto.Signature) bool {
	tx.Signature = sig
	return true
}

type VoteTx struct {
	Vote      Vote             `json:"vote"`
	Signature crypto.Signature `json:"signature"`
}

func (tx *VoteTx) SignBytes() []byte { return tx.Vote.SignBytes() }
func (tx *VoteTx) SetSignature(pub crypto.PubKey, sig crypto.Signature) bool {
	tx.Signature = sig
	return true
}

type Tx interface {
	SignBytes() []byte
	SetSignature(crypto.PubKey, crypto.Signature) bool
}

const (
	TxTypeProposal = byte(0x01)
	TxTypeVote     = byte(0x02)
)

var _ = wire.RegisterInterface(
	struct{ Tx }{},
	wire.ConcreteType{&ProposalTx{}, TxTypeProposal},
	wire.ConcreteType{&VoteTx{}, TxTypeVote},
)

//----------------------------------------

type GovMeta struct {
	Height       uint64 // The current block height
	NumEntities  int    // For EntityAddr generation
	NumGroups    int    // For GroupID generation
	NumProposals int    // For ProposalID generation
}

//----------------------------------------

func EntityKey(entityAddr []byte) []byte {
	return append([]byte("gov/e/"), entityAddr...)
}

func GroupKey(groupID string) []byte {
	return []byte("gov/g/" + groupID)
}

func ActiveProposalKey(proposalID string) []byte {
	return []byte("gov/ap/" + proposalID)
}

func GovMetaKey() []byte {
	return []byte("gov/meta")
}
