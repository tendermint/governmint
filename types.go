package governmint

import (
	"bytes"
	"fmt"
	"time"

	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-merkle"
	"github.com/tendermint/go-wire"
)

//-------------
// Entities, Groups, and Proposals are Get/Set/Rm able on Governmint by their ID()
// They are GovernmintObjects

type GovernmintObject interface {
}

const (
	objTypeEntity   = byte(0x01)
	objTypeGroup    = byte(0x02)
	objTypeProposal = byte(0x03)
)

var _ = wire.RegisterInterface(
	struct{ GovernmintObject }{},
	wire.ConcreteType{&Entity{}, objTypeEntity},
	wire.ConcreteType{&Group{}, objTypeGroup},
	wire.ConcreteType{&Proposal{}, objTypeProposal},
)

type Entity struct {
	Name      string        `json:"name"` // Unique
	PubKey    crypto.PubKey `json:"pub_key"`
	InvitedBy string        `json:"invited_by"`
	Invites   int           `json:"invites"`
}

func (e *Entity) ID() string {
	return e.Name
}

type Member struct {
	EntityID    string `json:"entity_id"`
	VotingPower int    `json:"voting_power"`
}

type Group struct {
	Name        string    `json:"name"` // Unique
	Version     int       `json:"version"`
	LastUpdated time.Time `json:"last_updates"`
	Members     []*Member `json:"members"`
}

func (g *Group) ID() string {
	return g.Name
}

type Proposal struct {
	*ProposalTx

	Votes        []Vote // same order as Group.Members
	votesFor     int
	votesAgainst int
}

func (p *Proposal) ID() string {
	return p.ProposalTx.Name
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
	wire.WriteJSON(o, buf, &n, &err)
	return merkle.SimpleHashFromBinary(buf.Bytes())
}

func SignBytes(tx Tx) []byte {
	buf := new(bytes.Buffer)
	var err error
	var n int
	wire.WriteJSON(tx, buf, &n, &err)
	return buf.Bytes()
}

type ProposalTx struct {
	Name       string `json:"name"`
	Data       string `json:"data"`
	GroupID    string `json:"group"`
	ProposerID string `json:"proposer"`
}

type VoteTx struct {
	ProposalID string `json:"proposal"`
	Vote       bool   `json:"vote"`
	Member     int    `json:"member"` // member's position in the group
}

func (tx *VoteTx) String() string {
	return fmt.Sprintf("%X %v %d", tx.ProposalID, tx.Vote, tx.Member)

}
