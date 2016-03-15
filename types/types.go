package types

import (
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
)

const (
	AdminGroupID      = "admin"
	ValidatorsGroupID = "validators"
)

type Entity struct {
	ID     string        `json:"id"`
	PubKey crypto.PubKey `json:"pub_key"`
}

type Group struct {
	ID       string   `json:"id"`
	ParentID string   `json:"parent_id"`
	Version  int      `json:"version"`
	Members  []Member `json:"members"`
}

type Member struct {
	EntityID    string `json:"entity_id"`
	VotingPower uint64 `json:"voting_power"`
}

func NewMember(entityID string, votingPower uint64) Member {
	return Member{entityID, votingPower}
}

type Vote struct {
	Height     uint64 `json:"height"`
	EntityID   string `json:"entity_id"`
	ProposalID string `json:"proposal_id"`
	Value      string `json:"value"`
}

type SignedVote struct {
	Vote      Vote             `json:"vote"`
	Signature crypto.Signature `json:"signature"`
}

func NewSignedVote(vote Vote, sig crypto.Signature) SignedVote {
	return SignedVote{vote, sig}
}

func (vote Vote) SignBytes() []byte { return wire.JSONBytes(vote) }

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
	EntityID  string           `json:"entity_id"`
	Proposal  Proposal         `json:"proposal"`
	Signature crypto.Signature `json:"signature"`
}

func (tx *ProposalTx) SignBytes() []byte { return tx.Proposal.SignBytes() }

type VoteTx struct {
	Vote      Vote             `json:"vote"`
	Signature crypto.Signature `json:"signature"`
}

func (tx *VoteTx) SignBytes() []byte { return tx.Vote.SignBytes() }

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

type GovMeta struct {
	Height       uint64 // The current block height
	NumEntities  int    // For EntityID generation
	NumGroups    int    // For GroupID generation
	NumProposals int    // For ProposalID generation
}

//----------------------------------------

func EntityKey(entityID string) []byte {
	return []byte("GOV:e:" + entityID)
}

func GroupKey(groupID string) []byte {
	return []byte("GOV:g:" + groupID)
}

func ActiveProposalKey(proposalID string) []byte {
	return []byte("GOV:ap:" + proposalID)
}

func GovMetaKey() []byte {
	return []byte("GOV:meta")
}

//----------------------------------------

type TMSPError struct {
	Code tmsp.CodeType
	Log  string
}

func (tmspErr TMSPError) IsOK() bool {
	return tmspErr.Code == tmsp.CodeType_OK
}
