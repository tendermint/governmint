package tests

import (
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/governmint/types"
)

// TODO move this to gov/testutil package

func SignVote(secret string, vote types.Vote) crypto.Signature {
	privKey := crypto.GenPrivKeyEd25519FromSecret([]byte(secret))
	voteSignBytes := vote.SignBytes()
	return privKey.Sign(voteSignBytes)
}

func SignProposal(secret string, proposal types.Proposal) crypto.Signature {
	privKey := crypto.GenPrivKeyEd25519FromSecret([]byte(secret))
	proposalSignBytes := proposal.SignBytes()
	return privKey.Sign(proposalSignBytes)
}

func VoteTx(secret string, height uint64,
	proposalID string, value string) *types.VoteTx {
	vote := types.Vote{
		Height:     height,
		EntityID:   EntityID(secret),
		ProposalID: proposalID,
		Value:      value,
	}
	return &types.VoteTx{
		Vote:      vote,
		Signature: SignVote(secret, vote),
	}
}

func ProposalTx(secret string, proposalID string, voteGroupID string,
	start uint64, end uint64, info types.ProposalInfo) *types.ProposalTx {

	proposal := types.Proposal{
		ID:          proposalID,
		VoteGroupID: voteGroupID,
		StartHeight: start,
		EndHeight:   end,
		Info:        info,
	}
	return &types.ProposalTx{
		EntityID:  EntityID(secret),
		Proposal:  proposal,
		Signature: SignProposal(secret, proposal),
	}
}

type PrivEntity struct {
	types.Entity
	PrivKey crypto.PrivKey
}

// By testing convention, entities have ids "id(SECRET)"
func Entities(secrets []string) []PrivEntity {
	entities := make([]PrivEntity, len(secrets))
	for i, secret := range secrets {
		privKey := crypto.GenPrivKeyEd25519FromSecret([]byte(secret))
		entities[i] = PrivEntity{
			Entity: types.Entity{
				ID:     Fmt("id(%v)", secret),
				PubKey: privKey.PubKey(),
			},
			PrivKey: privKey,
		}
	}
	return entities
}

func Members(secrets []string, power uint64) []types.Member {
	members := make([]types.Member, len(secrets))
	for i, secret := range secrets {
		members[i] = types.Member{
			EntityID:    Fmt("id(%v)", secret),
			VotingPower: power,
		}
	}
	return members
}

func EntityID(secret string) string {
	return Fmt("id(%v)", secret)
}
