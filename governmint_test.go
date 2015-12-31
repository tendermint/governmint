package governmint

import (
	"testing"
)

//-------------------------------------------------------------

func TestProposalTx(t *testing.T) {
	gov := newGovernmint()

	// invalid group
	badProp := &ProposalTx{
		Data:       "bernie sanders for pres",
		GroupID:    gov.group.ID() + "-bad",
		ProposerID: gov.group.Members[0].EntityID,
	}
	sig := gov.keys[0].Sign(SignBytes(badProp))
	if retCode := gov.gov.addProposal(badProp, sig); retCode == 0 {
		t.Fatal("expected addProposal to fail")
	}

	// invalid sig
	sig = gov.keys[0].Sign(append(SignBytes(badProp), byte(1)))
	if retCode := gov.gov.addProposal(badProp, sig); retCode == 0 {
		t.Fatal("expected addProposal to fail")
	}

	// a good proposal to add twice
	goodProp := &ProposalTx{
		Data:       "bernie sanders for pres",
		GroupID:    gov.group.ID(),
		ProposerID: gov.group.Members[0].EntityID,
	}
	sig = gov.keys[0].Sign(SignBytes(goodProp))
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
		GroupID:    gov.group.ID(),
		ProposerID: gov.group.Members[0].EntityID,
	}
	propId := string(Hash(propTx))
	sig := gov.keys[0].Sign(SignBytes(propTx))
	if retCode := gov.gov.addProposal(propTx, sig); retCode != 0 {
		t.Fatal("expected addProposal to pass")
	}

	// add some votes
	vote0 := newVoteTx(propId, true, 0)
	sig = gov.keys[0].Sign(SignBytes(vote0))
	if retCode := gov.gov.addVote(vote0, sig); retCode != 0 {
		t.Fatal("expected addVote to pass")
	}

	vote1 := newVoteTx(propId, false, 1)
	sig = gov.keys[1].Sign(SignBytes(vote1))
	if retCode := gov.gov.addVote(vote1, sig); retCode != 0 {
		t.Fatal("expected addVote to pass")
	}

	vote2 := newVoteTx(propId, true, 2)
	sig = gov.keys[2].Sign(SignBytes(vote2))
	if retCode := gov.gov.addVote(vote2, sig); retCode != 0 {
		t.Fatal("expected addVote to pass")
	}

	// the proposal should be resolved
	res := gov.gov.GetResolution(propId)
	if res == nil {
		t.Fatal("expected proposal to become a resolution")
	}
}
