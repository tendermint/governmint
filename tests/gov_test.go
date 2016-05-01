package tests

import (
	base "github.com/tendermint/basecoin/types"
	gm "github.com/tendermint/governmint/gov"
	govutil "github.com/tendermint/governmint/testutil"
	"github.com/tendermint/governmint/types"
	tmsputil "github.com/tendermint/tmsp/testutil"
	tmsp "github.com/tendermint/tmsp/types"
	"testing"
)

func TestIntegration(t *testing.T) {

	gov := gm.NewGovernmint()
	store := base.NewMemKVStore()
	t.Log("Created gov. ", gov)

	gov.InitChain(store, []*tmsp.Validator{
		tmsputil.Validator("entity1", 1),
		tmsputil.Validator("entity2", 1),
		tmsputil.Validator("entity3", 1),
	})

	res := gov.RunTxParsed(store, govutil.ProposalTx("secret1",
		"my_proposal_id", "my_vote_group_id", 0, 1,
		&types.GroupCreateProposalInfo{
			NewGroupID: "new_group_id",
			Members:    govutil.Members([]string{"entity1"}, 1),
		},
	))

	t.Log(res.Code, res.Data, res.Log)
}
