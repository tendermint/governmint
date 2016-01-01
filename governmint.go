package governmint

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
	"sync"

	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-merkle"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/tmsp/types"
)

//----------------------------------------
// prefixes for putting everything in one merkle tree

func PrefixEntityKey(id string) string {
	return "entity-" + id
}

func PrefixGroupKey(id string) string {
	return "group-" + id
}

func PrefixProposalKey(id string) string {
	return "proposal-" + id // note these may not be legible
}

func PrefixResolutionKey(id string) string {
	return "resolution-" + id // note these may not be legible
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
	return &Governmint{state: state}
}

func (g *Governmint) Copy() *Governmint {
	return &Governmint{
		state: g.state.Copy(),
	}
}

// Entities
func (g *Governmint) GetEntity(id string) *Entity {
	_, v := g.state.Get(PrefixEntityKey(id))
	if v == nil {
		return nil
	}

	return v.(*Entity) //
}

func (g *Governmint) SetEntity(id string, e *Entity) {
	g.state.Set(PrefixEntityKey(id), e)
}

func (g *Governmint) RmEntity(id string) {
	g.state.Remove(PrefixEntityKey(id))
}

// Groups
func (g *Governmint) GetGroup(id string) *Group {
	_, v := g.state.Get(PrefixGroupKey(id))
	if v == nil {
		return nil
	}

	return v.(*Group) //
}

func (g *Governmint) SetGroup(id string, gr *Group) {
	g.state.Set(PrefixGroupKey(id), gr)
}

func (g *Governmint) RmGroup(id string) {
	g.state.Remove(PrefixGroupKey(id))
}

// Proposals
func (g *Governmint) GetProposal(id string) *Proposal {
	_, v := g.state.Get(PrefixProposalKey(id))
	if v == nil {
		return nil
	}

	return v.(*Proposal) //
}

func (g *Governmint) SetProposal(id string, p *Proposal) {
	g.state.Set(PrefixProposalKey(id), p)
}

func (g *Governmint) RmProposal(id string) {
	g.state.Remove(PrefixProposalKey(id))
}

// Resolutions (closed proposals)
func (g *Governmint) GetResolution(id string) *Proposal {
	_, v := g.state.Get(PrefixResolutionKey(id))
	if v == nil {
		return nil
	}

	return v.(*Proposal) //
}

func (g *Governmint) SetResolution(id string, p *Proposal) {
	g.state.Set(PrefixResolutionKey(id), p)
}

func (g *Governmint) RmResolution(id string) {
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
	log.Notice("Loaded state from file", "hash", hash)
	return gov, nil
}

//----------------------------------------------------------------
// tx processing

func (gov *Governmint) addProposal(tx *ProposalTx, sig crypto.Signature) types.RetCode {

	// check sig
	m := gov.GetEntity(tx.ProposerID)
	if m == nil {
		log.Debug("Unknown proposer", "id", tx.ProposerID)
		return types.RetCodeUnauthorized
	}
	if !m.PubKey.VerifyBytes(SignBytes(tx), sig) {
		log.Debug("Invalid signature", "signbytes", SignBytes(tx), "pub", m.PubKey)
		return types.RetCodeUnauthorized
	}

	p := &Proposal{ProposalTx: tx}
	id := p.ID()

	if p2 := gov.GetProposal(id); p2 != nil {
		log.Debug("Proposal already exists", "id", id)
		return types.RetCodeUnauthorized //fmt.Errorf("Proposal already exists")
	}

	if r := gov.GetResolution(id); r != nil {
		log.Debug("Proposal has already been resolved", "id", id)
		return types.RetCodeUnauthorized //fmt.Errorf("Proposal already exists")
	}

	var group *Group
	if group = gov.GetGroup(tx.GroupID); group == nil {
		log.Debug("Group does not exist", "id", tx.GroupID)
		return types.RetCodeUnauthorized //fmt.Errorf("Group does not exist")
	}

	p.Votes = make([]Vote, len(group.Members))
	gov.SetProposal(id, p)
	log.Notice(Fmt("Added proposal %X", id))
	return types.RetCodeOK
}

func (gov *Governmint) addVote(tx *VoteTx, sig crypto.Signature) types.RetCode {
	p := gov.GetProposal(tx.ProposalID)
	if p == nil {
		log.Debug("Proposal does not exist", "id", tx.ProposalID)
		return types.RetCodeUnauthorized //fmt.Errorf("Proposal does not exist")
	}

	gr := gov.GetGroup(p.GroupID)

	if tx.Member > len(gr.Members) {
		log.Debug("Invalid member index", "index", tx.Member)
		return types.RetCodeUnauthorized //fmt.Errorf("Invalid member index")
	}

	// check sig
	m := gr.Members[tx.Member]
	if m == nil {
		log.Debug("Invalid member initialization", "index", tx.Member)
		return types.RetCodeUnauthorized
	}
	entity := gov.GetEntity(m.EntityID)
	if !entity.PubKey.VerifyBytes(SignBytes(tx), sig) {
		log.Debug("Invalid signawture")
		return types.RetCodeUnauthorized
	}

	p.Votes[tx.Member] = Vote{tx.Vote, sig}
	if tx.Vote {
		p.votesFor += 1
		if p.votesFor > len(p.Votes)/2 {
			gov.RmProposal(tx.ProposalID)
			gov.SetResolution(tx.ProposalID, p)
			log.Notice("Proposal -> Resolution", "id", p.ID())
		}
	} else {
		p.votesAgainst += 1
		if p.votesAgainst > len(p.Votes)/2 {
			gov.RmProposal(tx.ProposalID)
			gov.SetResolution(tx.ProposalID, p)
			log.Notice("Proposal Vetoed", "id", p.ID())
		}
	}
	gov.SetProposal(tx.ProposalID, p)
	log.Notice(Fmt("Added vote %v", tx))
	return types.RetCodeOK
}
