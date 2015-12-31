package governmint

import (
	"fmt"

	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-merkle"
	"github.com/tendermint/go-wire"
)

//-------------------------------------------------------------
// utility functions

func newEntity(i int) (*Entity, crypto.PrivKey) {
	key := crypto.GenPrivKeyEd25519()
	entity := &Entity{
		Name:   fmt.Sprintf("entity%d", i),
		PubKey: key.PubKey(),
	}
	return entity, key
}

func createEntities(n int) ([]*Entity, []crypto.PrivKey) {
	entities := make([]*Entity, n)
	keys := make([]crypto.PrivKey, n)
	for i := 0; i < n; i++ {
		entities[i], keys[i] = newEntity(i)
	}
	return entities, keys
}

type testEntity struct {
	entity *Entity
	key    crypto.PrivKey
}

func membersFromEntities(entities []*Entity) []*Member {
	members := make([]*Member, len(entities))
	for i, e := range entities {
		members[i] = &Member{e.ID(), 1}
	}
	return members
}

type testGovernmint struct {
	gov      *Governmint
	entities []*Entity
	keys     []crypto.PrivKey
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
	entities, keys := createEntities(3)
	for _, e := range entities {
		gov.SetEntity(e.ID(), e)
	}

	// add a group
	group := &Group{
		Name:    "the_group",
		Members: membersFromEntities(entities),
	}
	gov.SetGroup(Hash(group), group)

	return testGovernmint{gov, entities, keys, group}
}

func newVoteTx(propId []byte, vote bool, member int) *VoteTx {
	return &VoteTx{
		ProposalID: propId,
		Vote:       vote,
		Member:     member,
	}
}
