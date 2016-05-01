package gov

import (
	base "github.com/tendermint/basecoin/types"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/governmint/types"
	"testing"
)

func TestUnit(t *testing.T) {
	gov := NewGovernmint()

	// Test Entity
	{
		store := base.NewMemKVStore()
		privKey := crypto.GenPrivKeyEd25519()
		pubKey := privKey.PubKey()

		gov.SetEntity(store, &types.Entity{
			Addr:   []byte("my_entity_id"),
			PubKey: pubKey,
		})

		entityCopy, ok := gov.GetEntity(store, []byte("my_entity_id"))
		if !ok {
			t.Error("Saved(set) entity does not exist")
		}
		if string(entityCopy.Addr) != "my_entity_id" {
			t.Error("Got wrong entity id")
		}
		if !pubKey.Equals(entityCopy.PubKey) {
			t.Error("Got wrong entity pubkey")
		}

		entityBad, ok := gov.GetEntity(store, []byte("my_bad_id"))
		if ok || entityBad != nil {
			t.Error("Expected nil entity")
		}
	}

	// Test Group
	{
		store := base.NewMemKVStore()
		gov.SetGroup(store, &types.Group{
			ID:      "my_group_id",
			Version: 1,
			Members: []types.Member{
				types.Member{
					EntityAddr:  []byte("my_entity_id"),
					VotingPower: 1,
				},
			},
		})

		groupCopy, ok := gov.GetGroup(store, "my_group_id")
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
		if string(groupCopy.Members[0].EntityAddr) != "my_entity_id" {
			t.Error("Group member's entity id is wrong")
		}

		groupBad, ok := gov.GetGroup(store, "my_bad_id")
		if ok || groupBad != nil {
			t.Error("Expected nil group")
		}
	}

	// Test ActiveProposal
	{
		store := base.NewMemKVStore()
		ap := &types.ActiveProposal{
			Proposal: types.Proposal{
				ID:          "my_proposal_id",
				VoteGroupID: "my_vote_group_id",
				Info: &types.GroupUpdateProposalInfo{
					UpdateGroupID: "my_update_group_id",
					NextVersion:   1,
					ChangedMembers: []types.Member{
						types.Member{
							EntityAddr:  []byte("entity1"),
							VotingPower: 1,
						},
					},
				},
				StartHeight: 99,
				EndHeight:   100,
			},
			SignedVotes: []types.SignedVote{
				types.SignedVote{
					Vote: types.Vote{
						Height:     123,
						EntityAddr: []byte("entity1"),
						ProposalID: "my_proposal_id",
						Value:      "my_vote",
					},
					Signature: nil, // TODO set a sig
				},
			},
		}
		gov.SetActiveProposal(store, ap)
		proposalID := ap.Proposal.ID

		apCopy, ok := gov.GetActiveProposal(store, proposalID)
		if !ok {
			t.Error("Saved(set) ap does not exist")
		}
		if apCopy.Proposal.VoteGroupID != "my_vote_group_id" {
			t.Error("Got wrong ap proposal vote group id")
		}
		if apCopy.Proposal.Info.(*types.GroupUpdateProposalInfo).UpdateGroupID != "my_update_group_id" {
			t.Error("Got wrong ap proposal update group id")
		}
		if apCopy.Proposal.Info.(*types.GroupUpdateProposalInfo).NextVersion != 1 {
			t.Error("Got wrong ap proposal group version ")
		}
		if len(apCopy.Proposal.Info.(*types.GroupUpdateProposalInfo).ChangedMembers) != 1 {
			t.Error("Got wrong ap proposal changed members size")
		}
		if len(apCopy.SignedVotes) != 1 {
			t.Error("Got wrong ap proposal votes size")
		}

		apBad, ok := gov.GetActiveProposal(store, "my_bad_id")
		if ok || apBad != nil {
			t.Error("Expected nil ap")
		}
	}
}
