package governmint

import (
	"fmt"
	"testing"

	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-merkle"
	"github.com/tendermint/go-wire"
)

//-------------------------------------------------------------
// utility functions

func newEntity(i int) testEntity {
	key := crypto.GenPrivKeyEd25519()
	entity := &Entity{
		Name:   fmt.Sprintf("entity%d", i),
		PubKey: key.PubKey(),
	}
	return testEntity{Hash(*entity), entity, key}
}

func createEntities(n int) []testEntity {
	entities := make([]testEntity, n)
	for i := 0; i < n; i++ {
		entities[i] = newEntity(i)
	}
	return entities
}

type testEntity struct {
	id     []byte
	entity *Entity
	key    crypto.PrivKey
}

func membersFromEntities(entities []testEntity) []*Member {
	members := make([]*Member, len(entities))
	for i, e := range entities {
		members[i] = &Member{e.id, 1}
	}
	return members
}

type testGovernmint struct {
	gov      *Governmint
	entities []testEntity
	group    *Group
}

func newGovernmint() testGovernmint {
	state := merkle.NewIAVLTree(
		wire.BasicCodec,
		wire.BasicCodec,
		0,
		nil,
	)

	gov := &Governmint{
		state,
	}

	// add the entities
	entities := createEntities(3)
	for _, e := range entities {
		gov.SetEntity(e.id, e.entity)
	}

	// add a group
	group := &Group{
		Name:    "the_group",
		Members: membersFromEntities(entities),
	}
	gov.SetGroup(Hash(group), group)

	return testGovernmint{gov, entities, group}
}

func newVoteTx(propId []byte, vote bool, member int) *VoteTx {
	return &VoteTx{
		ProposalID: propId,
		Vote:       vote,
		Member:     member,
	}
}

//-------------------------------------------------------------

func TestProposalTx(t *testing.T) {
	gov := newGovernmint()

	// invalid group
	badProp := &ProposalTx{
		Data:       "bernie sanders for pres",
		GroupID:    append(Hash(gov.group), byte(1)),
		ProposerID: gov.group.Members[0].EntityID,
	}
	sig := gov.entities[0].key.Sign(SignBytes(badProp))
	if retCode := gov.gov.addProposal(badProp, sig); retCode == 0 {
		t.Fatal("expected addProposal to fail")
	}

	// invalid sig
	sig = gov.entities[0].key.Sign(append(SignBytes(badProp), byte(1)))
	if retCode := gov.gov.addProposal(badProp, sig); retCode == 0 {
		t.Fatal("expected addProposal to fail")
	}

	// a good proposal to add twice
	goodProp := &ProposalTx{
		Data:       "bernie sanders for pres",
		GroupID:    Hash(gov.group),
		ProposerID: gov.group.Members[0].EntityID,
	}
	sig = gov.entities[0].key.Sign(SignBytes(goodProp))
	if retCode := gov.gov.addProposal(goodProp, sig); retCode != 0 {
		t.Fatal("expected addProposal to pass")
	}
	if retCode := gov.gov.addProposal(goodProp, sig); retCode == 0 {
		t.Fatal("expected addProposal to fail")
	}
}

func TestVoteTx(t *testing.T) {
	gov := newGovernmint()

	// add a proposal
	propTx := &ProposalTx{
		Data:       "bernie sanders for pres",
		GroupID:    Hash(gov.group),
		ProposerID: gov.group.Members[0].EntityID,
	}
	propId := TxID(propTx)
	sig := gov.entities[0].key.Sign(SignBytes(propTx))
	if retCode := gov.gov.addProposal(propTx, sig); retCode != 0 {
		t.Fatal("expected addProposal to pass")
	}

	// add some votes
	vote0 := newVoteTx(propId, true, 0)
	sig = gov.entities[0].key.Sign(SignBytes(vote0))
	if retCode := gov.gov.addVote(vote0, sig); retCode != 0 {
		t.Fatal("expected addVote to pass")
	}

	vote1 := newVoteTx(propId, false, 1)
	sig = gov.entities[1].key.Sign(SignBytes(vote1))
	if retCode := gov.gov.addVote(vote1, sig); retCode != 0 {
		t.Fatal("expected addVote to pass")
	}

	vote2 := newVoteTx(propId, true, 2)
	sig = gov.entities[2].key.Sign(SignBytes(vote2))
	if retCode := gov.gov.addVote(vote2, sig); retCode != 0 {
		t.Fatal("expected addVote to pass")
	}

	// the proposal should be resolved
	res := gov.gov.GetResolution(propId)
	if res == nil {
		t.Fatal("expected proposal to become a resolution")
	}
}
