package gov

import (
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/governmint/types"
	eyesApp "github.com/tendermint/merkleeyes/app"
	eyes "github.com/tendermint/merkleeyes/client"
	"github.com/tendermint/tmsp/server"
	"testing"
)

func makeMerkleEyesServer(addr string) *server.Server {
	app := eyesApp.NewMerkleEyesApp()
	s, err := server.NewServer(addr, app)
	if err != nil {
		panic("starting MerkleEyes listener: " + err.Error())
	}
	return s
}

func makeMerkleEyesClient(addr string) *eyes.Client {
	c, err := eyes.NewClient("unix://test.sock")
	if err != nil {
		panic("creating MerkleEyes client: " + err.Error())
	}
	return c
}

func TestUnit(t *testing.T) {
	s := makeMerkleEyesServer("unix://test.sock")
	defer s.Stop()
	c := makeMerkleEyesClient("unix://test.sock")
	defer c.Stop()
	gov := NewGovernmint(c)

	// Test Entity
	{
		privKey := crypto.GenPrivKeyEd25519()
		pubKey := privKey.PubKey()

		gov.setEntity(&types.Entity{
			ID:     "my_entity_id",
			PubKey: pubKey,
		})

		entityCopy, ok := gov.getEntity("my_entity_id")
		if !ok {
			t.Error("Saved(set) entity does not exist")
		}
		if entityCopy.ID != "my_entity_id" {
			t.Error("Got wrong entity id")
		}
		if !pubKey.Equals(entityCopy.PubKey) {
			t.Error("Got wrong entity pubkey")
		}

		entityBad, ok := gov.getEntity("my_bad_id")
		if ok || entityBad != nil {
			t.Error("Expected nil entity")
		}
	}

	// Test Group
	{
		gov.setGroup(&types.Group{
			ID:      "my_group_id",
			Version: 1,
			Members: []types.Member{
				types.Member{
					EntityID:    "my_entity_id",
					VotingPower: 1,
				},
			},
		})

		groupCopy, ok := gov.getGroup("my_group_id")
		if !ok {
			t.Error("Saved(set) group does not exist")
		}
		if groupCopy.ID != "my_group_id" {
			t.Error("Got wrong group id")
		}
		if groupCopy.Version != 1 {
			t.Error("Got wrong group version ")
		}
		if len(groupCopy.Members) != 1 {
			t.Error("Got wrong group members size")
		}
		if groupCopy.Members[0].EntityID != "my_entity_id" {
			t.Error("Group member's entity id is wrong")
		}

		groupBad, ok := gov.getGroup("my_bad_id")
		if ok || groupBad != nil {
			t.Error("Expected nil group")
		}
	}

	// Test ActiveProposal
	{
		ap := &types.ActiveProposal{
			Proposal: &types.GroupUpdateProposal{
				GroupID:      "my_group_id",
				GroupVersion: 1,
				AddMembers: []types.Member{
					types.Member{
						EntityID:    "entity1",
						VotingPower: 1,
					},
				},
				RemMembers: []types.Member{
					types.Member{
						EntityID:    "entity2",
						VotingPower: 1,
					},
				},
			},
			Votes: []*types.Vote{
				&types.Vote{
					Value:     "my_vote",
					Signature: nil, // TODO set a sig
				},
			},
		}
		gov.setActiveProposal(ap)
		proposalID := types.ProposalID(ap.Proposal)

		apCopy, ok := gov.getActiveProposal(proposalID)
		if !ok {
			t.Error("Saved(set) ap does not exist")
		}
		if apCopy.Proposal.(*types.GroupUpdateProposal).GroupID != "my_group_id" {
			t.Error("Got wrong ap proposal group id")
		}
		if apCopy.Proposal.(*types.GroupUpdateProposal).GroupVersion != 1 {
			t.Error("Got wrong ap proposal group version ")
		}
		if len(apCopy.Proposal.(*types.GroupUpdateProposal).AddMembers) != 1 {
			t.Error("Got wrong ap proposal add members size")
		}
		if len(apCopy.Proposal.(*types.GroupUpdateProposal).RemMembers) != 1 {
			t.Error("Got wrong ap proposal add members size")
		}
		if len(apCopy.Votes) != 1 {
			t.Error("Got wrong ap proposal votes size")
		}

		apBad, ok := gov.getActiveProposal("my_bad_id")
		if ok || apBad != nil {
			t.Error("Expected nil ap")
		}
	}
}
