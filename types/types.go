package types

import (
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
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
	ID      string   `json:"id"`
	Version int      `json:"version"`
	Members []Member `json:"members"`
}

type Member struct {
	EntityID    string `json:"entity_id"`
	VotingPower uint64 `json:"voting_power"`
}

func NewMember(entityID string, votingPower int) Member {
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

type Proposal struct {
	ID          string       `json:"id"`
	Info        ProposalInfo `json:"info"`
	StartHeight uint64       `json:"start_height"`
	EndHeight   uint64       `json:"end_height"`
}

type ActiveProposal struct {
	Proposal    `json:"proposal"`
	SignedVotes []SignedVote `json:"signed_votes"`
}

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

const GovMetaKey = "GOV:meta"

//----------------------------------------

type GroupCreateProposalInfo struct {
	GroupID string   `json:"group_id"`
	Members []Member `json:"members"`
}

func (p GroupCreateProposalInfo) GetGroupID() string { return p.GroupID }

type GroupUpdateProposalInfo struct {
	GroupID        string   `json:"group_id"`
	GroupVersion   int      `json:"group_version"`
	ChangedMembers []Member `json:"changed_members"` // 0 VotingPower to remove
}

func (p GroupUpdateProposalInfo) GetGroupID() string { return p.GroupID }

type TextProposalInfo struct {
	GroupID string `json:"group_id"`
	Text    string `json:"text"`
}

func (p TextProposalInfo) GetGroupID() string { return p.GroupID }

type UpgradeProposalInfoModule struct {
	Name   string `json:"module"`
	Script string `json:"script"`
}

type UpgradeProposalInfo struct {
	Modules []UpgradeProposalInfoModule
}

func (p UpgradeProposalInfo) GetGroupID() string { return AdminGroupID }

type ProposalInfo interface {
	AssertIsProposalInfo()
	GetGroupID() string
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

func (tx *ProposalTx) SignBytes() []byte {
	return wire.JSONBytes(tx.Proposal)
}

type VoteTx struct {
	Vote      Vote             `json:"vote"`
	Signature crypto.Signature `json:"signature"`
}

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
