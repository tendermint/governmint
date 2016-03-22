package tests

import (
	gm "github.com/tendermint/governmint/gov"
	govutil "github.com/tendermint/governmint/testutil"
	"github.com/tendermint/governmint/types"
	eyesutil "github.com/tendermint/merkleeyes/testutil"
	tmsputil "github.com/tendermint/tmsp/testutil"
	tmsp "github.com/tendermint/tmsp/types"
	"testing"
)

func TestIntegration(t *testing.T) {

	svr, cli := eyesutil.CreateEyes(t)
	defer svr.Stop()
	defer cli.Stop()

	gov := gm.NewGovernmint(cli)
	t.Log("Created gov. ", gov.Info())

	gov.InitChain([]*tmsp.Validator{
		tmsputil.Validator("entity1", 1),
		tmsputil.Validator("entity2", 1),
		tmsputil.Validator("entity3", 1),
	})

	res := gov.RunTx(govutil.ProposalTx("secret1",
		"my_proposal_id", "my_vote_group_id", 0, 1,
		&types.GroupCreateProposalInfo{
			NewGroupID: "new_group_id",
			Members:    govutil.Members([]string{"entity1"}, 1),
		},
	))

	t.Log(res.Code, res.Data, res.Log)
}
