package governmint

import (
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-merkle"
	"github.com/tendermint/tmsp/types"
)

// Governmint is a merkle tree of entities, groups, proposals
type Governmint struct {
	state merkle.Tree
}

func PrefixEntityKey(id []byte) []byte {
	return append([]byte("entity-"), id...)
}

func PrefixGroupKey(id []byte) []byte {
	return append([]byte("group-"), id...)
}

func PrefixProposalKey(id []byte) []byte {
	return append([]byte("proposal-"), id...)
}

func PrefixResolutionKey(id []byte) []byte {
	return append([]byte("resolution-"), id...)
}

// Entities
func (g *Governmint) GetEntity(id []byte) *Entity {
	_, v := g.state.Get(PrefixEntityKey(id))
	if v == nil {
		return nil
	}

	return v.(*Entity) //
}

func (g *Governmint) SetEntity(id []byte, e *Entity) {
	g.state.Set(PrefixEntityKey(id), e)
}

func (g *Governmint) RmEntity(id []byte) {
	g.state.Remove(PrefixEntityKey(id))
}

// Groups
func (g *Governmint) GetGroup(id []byte) *Group {
	_, v := g.state.Get(PrefixGroupKey(id))
	if v == nil {
		return nil
	}

	return v.(*Group) //
}

func (g *Governmint) SetGroup(id []byte, gr *Group) {
	g.state.Set(PrefixGroupKey(id), gr)
}

func (g *Governmint) RmGroup(id []byte) {
	g.state.Remove(PrefixGroupKey(id))
}

// Proposals
func (g *Governmint) GetProposal(id []byte) *Proposal {
	_, v := g.state.Get(PrefixProposalKey(id))
	if v == nil {
		return nil
	}

	return v.(*Proposal) //
}

func (g *Governmint) SetProposal(id []byte, p *Proposal) {
	g.state.Set(PrefixProposalKey(id), p)
}

func (g *Governmint) RmProposal(id []byte) {
	g.state.Remove(PrefixProposalKey(id))
}

// Resolutions (closed proposals)
func (g *Governmint) GetResolution(id []byte) *Proposal {
	_, v := g.state.Get(PrefixResolutionKey(id))
	if v == nil {
		return nil
	}

	return v.(*Proposal) //
}

func (g *Governmint) SetResolution(id []byte, p *Proposal) {
	g.state.Set(PrefixResolutionKey(id), p)
}

func (g *Governmint) RmResolution(id []byte) {
	g.state.Remove(PrefixResolutionKey(id))
}

//----------------------------------------------------------------

func (gov *GovernmintAppContext) addProposal(tx *ProposalTx, sig crypto.Signature) types.RetCode {

	// check sig
	m := gov.GetEntity(tx.Proposer)
	if m == nil {
		return types.RetCodeUnauthorized
	}
	if !m.PubKey.VerifyBytes(SignBytes(tx), sig) {
		return types.RetCodeUnauthorized
	}

	id := TxID(tx)
	var p *Proposal
	if p = gov.GetProposal(id); p != nil {
		return types.RetCodeUnauthorized //fmt.Errorf("Proposal already exists")
	}

	var group *Group
	if group = gov.GetGroup(p.Group); group == nil {
		return types.RetCodeUnauthorized //fmt.Errorf("Group does not exist")
	}

	prop := &Proposal{
		Data:  tx.Data,
		Group: tx.Group,
		Votes: make([]Vote, len(group.Members)),
	}
	gov.SetProposal(id, prop)
	return types.RetCodeOK
}

func (gov *GovernmintAppContext) addVote(tx *VoteTx, sig crypto.Signature) types.RetCode {
	p := gov.GetProposal(tx.Proposal)
	if p == nil {
		return types.RetCodeUnauthorized //fmt.Errorf("Proposal does not exist")
	}

	gr := gov.GetGroup(p.Group)

	if tx.Member > len(gr.Members) {
		return types.RetCodeUnauthorized //fmt.Errorf("Invalid member index")
	}

	// check sig
	m := gr.Members[tx.Member]
	if m == nil {
		return types.RetCodeUnauthorized
	}
	entity := gov.GetEntity(m.Entity)
	if !entity.PubKey.VerifyBytes(SignBytes(tx), sig) {
		return types.RetCodeUnauthorized
	}

	p.Votes[tx.Member] = Vote{tx.Vote, sig}
	if tx.Vote {
		p.votesFor += 1
		if p.votesFor > len(p.Votes)/2 {
			gov.RmProposal(tx.Proposal)
			gov.SetResolution(tx.Proposal, p)
		}
	} else {
		p.votesAgainst += 1
		if p.votesAgainst > len(p.Votes)/2 {
			gov.RmProposal(tx.Proposal)
			gov.SetResolution(tx.Proposal, p)
		}
	}
	return types.RetCodeOK
}
