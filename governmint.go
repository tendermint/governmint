package governmint

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-merkle"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/tmsp/types"
)

//----------------------------------------
// prefixes for putting everything in one merkle tree

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

func jsonEncoder(o interface{}, w io.Writer, n *int, err *error) {
	wire.WriteJSON(o, w, n, err)

}

func jsonDecoder(r io.Reader, n *int, err *error) interface{} {
	var govObj GovernmintObject
	jsonBytes, err2 := ioutil.ReadAll(r)
	if err2 != nil {
		*err = err2
		return nil
	}
	return wire.ReadJSON(govObj, jsonBytes, err)
}

func jsonComparator(o1 interface{}, o2 interface{}) int {
	// not used
	return 0
}

var JsonCodec = wire.Codec{
	Encode:  jsonEncoder,
	Decode:  jsonDecoder,
	Compare: jsonComparator,
}

//----------------------------------------

// Governmint is a merkle tree of entities, groups, proposals
type Governmint struct {
	state merkle.Tree
}

func NewGovernmint() *Governmint {
	state := merkle.NewIAVLTree(
		wire.BasicCodec,
		JsonCodec,
		0,
		nil,
	)
	return &Governmint{state}
}

func (g *Governmint) Copy() *Governmint {
	return &Governmint{
		state: g.state.Copy(),
	}
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
// governmint file

func loadGovFromFile(govFile string) (*Governmint, error) {
	file, err := os.Open(govFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// each line is an entity
	entities := []*Entity{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var e Entity
		var err error
		eI := wire.ReadJSON(e, scanner.Bytes(), &err)
		e = eI.(Entity)
		if err != nil {
			return nil, err
		}
		entities = append(entities, &e)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// add the entities to the tree, make base group, save
	gov := NewGovernmint()
	for _, e := range entities {
		gov.SetEntity(e.ID(), e)
	}
	group := &Group{
		Name:    "Guvnahs",
		Version: 0,
		Members: membersFromEntities(entities),
	}
	gov.SetGroup(group.ID(), group)
	hash := gov.state.Hash()
	fmt.Printf("Governmint state hash: %X\n", hash)
	return gov, nil
}

//----------------------------------------------------------------
// tx processing

func (gov *Governmint) addProposal(tx *ProposalTx, sig crypto.Signature) types.RetCode {

	// check sig
	m := gov.GetEntity(tx.ProposerID)
	if m == nil {
		return types.RetCodeUnauthorized
	}
	if !m.PubKey.VerifyBytes(SignBytes(tx), sig) {
		return types.RetCodeUnauthorized
	}

	p := &Proposal{ProposalTx: tx}
	id := p.ID()

	if p2 := gov.GetProposal(id); p2 != nil {
		return types.RetCodeUnauthorized //fmt.Errorf("Proposal already exists")
	}

	var group *Group
	if group = gov.GetGroup(tx.GroupID); group == nil {
		return types.RetCodeUnauthorized //fmt.Errorf("Group does not exist")
	}

	p.Votes = make([]Vote, len(group.Members))
	gov.SetProposal(id, p)
	return types.RetCodeOK
}

func (gov *Governmint) addVote(tx *VoteTx, sig crypto.Signature) types.RetCode {
	p := gov.GetProposal(tx.ProposalID)
	if p == nil {
		return types.RetCodeUnauthorized //fmt.Errorf("Proposal does not exist")
	}

	gr := gov.GetGroup(p.GroupID)

	if tx.Member > len(gr.Members) {
		return types.RetCodeUnauthorized //fmt.Errorf("Invalid member index")
	}

	// check sig
	m := gr.Members[tx.Member]
	if m == nil {
		return types.RetCodeUnauthorized
	}
	entity := gov.GetEntity(m.EntityID)
	if !entity.PubKey.VerifyBytes(SignBytes(tx), sig) {
		return types.RetCodeUnauthorized
	}

	p.Votes[tx.Member] = Vote{tx.Vote, sig}
	if tx.Vote {
		p.votesFor += 1
		if p.votesFor > len(p.Votes)/2 {
			gov.RmProposal(tx.ProposalID)
			gov.SetResolution(tx.ProposalID, p)
		}
	} else {
		p.votesAgainst += 1
		if p.votesAgainst > len(p.Votes)/2 {
			gov.RmProposal(tx.ProposalID)
			gov.SetResolution(tx.ProposalID, p)
		}
	}
	return types.RetCodeOK
}
