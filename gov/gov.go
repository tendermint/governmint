package gov

import (
	"bytes"
	base "github.com/tendermint/basecoin/types"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/governmint/types"
	eyes "github.com/tendermint/merkleeyes/client"
	tmsp "github.com/tendermint/tmsp/types"
)

const (
	Version        = "0.1"
	MaxVotingPower = 1<<53 - 1
)

type Governmint struct {
	*types.GovMeta
	eyesCli *eyes.Client
}

func NewGovernmint(eyesCli *eyes.Client) *Governmint {
	gov := &Governmint{
		GovMeta: &types.GovMeta{
			Height:       0,
			NumEntities:  0,
			NumGroups:    0,
			NumProposals: 0,
		},
		eyesCli: eyesCli,
	}
	if meta, ok := gov.GetGovMeta(); ok {
		gov.GovMeta = meta
	}
	return gov
}

func (gov *Governmint) Info() string {
	return "Governmint v" + Version
}

func (gov *Governmint) SetOption(key string, value string) (log string) {
	switch key {
	case "admin":
		// Read entity
		var entity = new(types.Entity)
		err := wire.ReadJSONBytes([]byte(value), entity)
		if err != nil {
			return "Error decoding admin entity: " + err.Error()
		}
		// Save entity
		gov.SetEntity(entity)
		// Construct a group for admin
		adminGroup := &types.Group{
			ID:      types.AdminGroupID,
			Version: 0,
		}
		adminGroup.Members = []types.Member{
			types.Member{
				EntityAddr:  entity.Addr,
				VotingPower: 1,
			},
		}
		// Save admin group
		gov.SetGroup(adminGroup)
		return "Success"
	case "entity":
		// Read entity
		var entity = new(types.Entity)
		err := wire.ReadJSONBytes([]byte(value), entity)
		if err != nil {
			return "Error decoding entity: " + err.Error()
		}
		// Save entity
		gov.SetEntity(entity)
	}
	return "Unrecognized governmint option key " + key
}

// Implements basecoin.Plugin
func (gov *Governmint) RunTx(ctx base.CallContext, txBytes []byte) tmsp.Result {
	var tx types.Tx
	err := wire.ReadBinaryBytes(txBytes, &tx)
	if err != nil {
		return tmsp.ErrEncodingError.SetLog(Fmt("Error parsing Governmint tx bytes: %v", err.Error()))
	}
	return gov.RunTxParsed(tx)
}

func (gov *Governmint) RunTxParsed(tx types.Tx) tmsp.Result {
	switch tx := tx.(type) {
	case *types.ProposalTx:
		return gov.RunProposalTx(tx)
	case *types.VoteTx:
		return gov.RunVoteTx(tx)
	default:
		PanicSanity("Unknown tx type")
		return tmsp.NewError(tmsp.CodeType_InternalError, "Unknown tx type")
	}
}

func (gov *Governmint) RunProposalTx(tx *types.ProposalTx) tmsp.Result {
	// Ensure that proposer exists
	entity, ok := gov.GetEntity(tx.EntityAddr)
	if !ok {
		return tmsp.NewError(tmsp.CodeType_GovUnknownEntity,
			Fmt("Entity %v unknown", tx.EntityAddr))
	}
	// Ensure signature is valid
	signBytes := tx.SignBytes()
	if !entity.PubKey.VerifyBytes(signBytes, tx.Signature) {
		return tmsp.NewError(tmsp.CodeType_Unauthorized,
			Fmt("Invalid signature"))
	}
	// Ensure that the proposal is valid
	tmspErr := gov.validateProposal(tx.Proposal, entity)
	if !tmspErr.IsOK() {
		return tmspErr
	}
	// Good! Create a new proposal
	proposal := tx.Proposal
	aProposal := &types.ActiveProposal{
		Proposal:    proposal,
		SignedVotes: nil,
	}
	gov.SetActiveProposal(aProposal)
	return tmsp.NewResultOK(nil, "Proposal created")
}

func (gov *Governmint) RunVoteTx(tx *types.VoteTx) tmsp.Result {
	// Ensure that voter exists
	entity, ok := gov.GetEntity(tx.Vote.EntityAddr)
	if !ok {
		return tmsp.NewError(tmsp.CodeType_GovUnknownEntity,
			Fmt("Entity %v unknown", tx.Vote.EntityAddr))
	}
	// Ensure signature is valid
	signBytes := tx.SignBytes()
	if !entity.PubKey.VerifyBytes(signBytes, tx.Signature) {
		return tmsp.NewError(tmsp.CodeType_Unauthorized,
			Fmt("Invalid signature"))
	}
	// Ensure that the proposal exists
	aProposal, ok := gov.GetActiveProposal(tx.Vote.ProposalID)
	if !ok {
		return tmsp.NewError(tmsp.CodeType_GovUnknownProposal,
			Fmt("Unknown proposal %v", tx.Vote.ProposalID))
	}
	// Ensure that the vote's height is <= current height
	if !(tx.Vote.Height <= gov.GovMeta.Height) {
		return tmsp.NewError(tmsp.CodeType_GovInvalidVote,
			Fmt("Vote height is invalid"))
	}
	// Ensure that the vote's height matches the proposal's range
	if !(aProposal.StartHeight <= tx.Vote.Height &&
		tx.Vote.Height <= aProposal.EndHeight) {
		return tmsp.NewError(tmsp.CodeType_GovInvalidVote,
			Fmt("Vote height is invalid"))
	}
	// Fetch the proposal's voting group
	voteGroup, ok := gov.GetGroup(aProposal.VoteGroupID)
	if !ok {
		return tmsp.NewError(tmsp.CodeType_GovUnknownGroup,
			Fmt("Vote group with id %v doesn't exist", aProposal.VoteGroupID))
	}
	// Ensure that the voter belongs to the voting group
	if !isMemberOf(voteGroup, entity.Addr) {
		return tmsp.NewError(tmsp.CodeType_GovInvalidMember,
			Fmt("Voter %v not a member of %v", entity.Addr, voteGroup.ID))
	}
	// Ensure that the voter hasn't already voted
	if exists, _ := hasVoted(aProposal, entity.Addr); exists {
		return tmsp.NewError(tmsp.CodeType_GovDuplicateVote,
			Fmt("Voter %v already voted", entity.Addr))
	}
	// Good! Add a SignedVote
	aProposal.SignedVotes = append(aProposal.SignedVotes, types.SignedVote{
		Vote:      tx.Vote,
		Signature: tx.Signature,
	})
	gov.SetActiveProposal(aProposal)
	return tmsp.NewResultOK(nil, "Vote added to ActiveProposal")
}

// TMSP::Query
func (gov *Governmint) Query(query []byte) tmsp.Result {
	return tmsp.OK.SetLog("Query not supported")
}

// TMSP::Commit
func (gov *Governmint) Commit() tmsp.Result {
	return tmsp.OK
}

// TMSP::InitChain
func (gov *Governmint) InitChain(validators []*tmsp.Validator) {
	// Construct a group of entities for the validators.
	vGroup := &types.Group{
		ID:      types.ValidatorsGroupID,
		Version: 0,
	}
	for _, validator := range validators {
		var pubKey crypto.PubKey
		err := wire.ReadBinaryBytes(validator.PubKey, &pubKey)
		if err != nil {
			PanicSanity("Error reading validator pubkey: " + err.Error())
		}
		// Create an entity with this validator
		entity := &types.Entity{
			Addr:   pubKey.Address(),
			PubKey: pubKey,
		}
		gov.SetEntity(entity)
		// Add as member
		member := types.Member{
			EntityAddr:  entity.Addr,
			VotingPower: validator.Power,
		}
		vGroup.Members = append(vGroup.Members, member)
	}
	// Save vGroup
	gov.SetGroup(vGroup)
}

// TMSP::BeginBlock
func (gov *Governmint) BeginBlock(height uint64) {
	gov.GovMeta.Height = height
	return
}

// TMSP::EndBlock
func (gov *Governmint) EndBlock(height uint64) (changedValidators []*tmsp.Validator) {
	// Persist GovMeta
	gov.SetGovMeta(gov.GovMeta)
	// Return changed validators
	// XXX
	return
}

//----------------------------------------

func (gov *Governmint) validateProposal(p types.Proposal, proposer *types.Entity) (res tmsp.Result) {
	// Ensure that the proposal is unique
	if _, exists := gov.GetActiveProposal(p.ID); exists {
		return tmsp.NewError(tmsp.CodeType_GovDuplicateProposal,
			Fmt("Proposal with id %v already exists", p.ID))
	}
	// Ensure that the voting group exists
	voteGroup, ok := gov.GetGroup(p.VoteGroupID)
	if !ok {
		return tmsp.NewError(tmsp.CodeType_GovUnknownGroup,
			Fmt("Vote group with id %v doesn't exist", p.VoteGroupID))
	}
	// Ensure that the proposer belongs to the voting group
	if !isMemberOf(voteGroup, proposer.Addr) {
		return tmsp.NewError(tmsp.CodeType_Unauthorized,
			Fmt("Proposer %X is not member of %v", proposer.Addr, voteGroup.ID))
	}
	// Type dependent checks
	switch pInfo := p.Info.(type) {
	case *types.GroupCreateProposalInfo:
		// Ensure that the group ID is not taken
		if _, exists := gov.GetGroup(pInfo.NewGroupID); exists {
			return tmsp.NewError(tmsp.CodeType_GovDuplicateGroup,
				Fmt("Group with id %v already exists", pInfo.NewGroupID))
		}
		// Ensure that the member entities are unique
		if ok, dupe := validateUniqueMembers(pInfo.Members); !ok {
			return tmsp.NewError(tmsp.CodeType_GovDuplicateMember,
				Fmt("Duplicate member %v", dupe))
		}
		// Ensure that the member voting powers are reasonable
		for _, member := range pInfo.Members {
			if member.VotingPower == 0 {
				return tmsp.NewError(tmsp.CodeType_GovInvalidVotingPower,
					Fmt("Member cannot have 0 voting power"))
			}
			if member.VotingPower > MaxVotingPower {
				return tmsp.NewError(tmsp.CodeType_GovInvalidVotingPower,
					Fmt("Member voting power too large"))
			}
		}
		// Ensure that all the entities exist
		entityAddrs := entityAddrsFromMembers(pInfo.Members)
		_, unknownEntityAddr := gov.loadEntities(entityAddrs)
		if unknownEntityAddr != nil {
			return tmsp.NewError(tmsp.CodeType_GovUnknownEntity,
				Fmt("Group creation with unknown entity %X", unknownEntityAddr))
		}
	case *types.GroupUpdateProposalInfo:
		// Ensure that the update group exists
		updateGroup, ok := gov.GetGroup(pInfo.UpdateGroupID)
		if !ok {
			return tmsp.NewError(tmsp.CodeType_GovUnknownGroup,
				Fmt("Group with id %v doesn't exist", pInfo.UpdateGroupID))
		}
		// Ensure that the update group's parent is the voting group
		if updateGroup.ParentID != voteGroup.ID {
			return tmsp.NewError(tmsp.CodeType_Unauthorized,
				Fmt("Voting group %v cannot update %v", voteGroup.ID, updateGroup.ID))
		}
		// Ensure that the member entities are unique
		if ok, dupe := validateUniqueMembers(pInfo.ChangedMembers); !ok {
			return tmsp.NewError(tmsp.CodeType_GovDuplicateMember,
				Fmt("Duplicate member %v", dupe))
		}
		// Ensure that the member voting powers are reasonable
		for _, member := range pInfo.ChangedMembers {
			if member.VotingPower == 0 {
				// This is fine, we're removing members.
			}
			if member.VotingPower > MaxVotingPower {
				return tmsp.NewError(tmsp.CodeType_GovInvalidVotingPower,
					Fmt("Member voting power too large"))
			}
		}
		// Ensure that all the entities exist
		entityAddrs := entityAddrsFromMembers(pInfo.ChangedMembers)
		_, unknownEntityAddr := gov.loadEntities(entityAddrs)
		if unknownEntityAddr != nil {
			return tmsp.NewError(tmsp.CodeType_GovUnknownEntity,
				Fmt("Group creation with unknown entity %X", unknownEntityAddr))
		}
	case *types.TextProposalInfo:
		// TODO text string validation, e.g. max length
	case *types.UpgradeProposalInfo:
		// Ensure that the group is admin.
		if voteGroup.ID != types.AdminGroupID {
			return tmsp.NewError(tmsp.CodeType_Unauthorized,
				Fmt("Upgrade proposals must be voted on by admin group"))
		}
		// Ensure that the number of modules is > 0.
		if len(pInfo.Modules) == 0 {
			return tmsp.NewError(tmsp.CodeType_EncodingError,
				Fmt("Software upgrade requires > 0 modules"))
		}
	}
	return tmsp.NewResultOK(nil, "")
}

// Returns (true, "") if members are unique
// Returns (false, <duplicateEntityAddr>) if members are not unique
// NOTE: zero members is fine.
func validateUniqueMembers(members []types.Member) (bool, []byte) {
	entityAddrs := map[string]struct{}{}
	for _, member := range members {
		if _, exists := entityAddrs[string(member.EntityAddr)]; exists {
			return false, member.EntityAddr
		}
		entityAddrs[string(member.EntityAddr)] = struct{}{}
	}
	return true, nil
}

func entityAddrsFromMembers(members []types.Member) [][]byte {
	entityAddrs := make([][]byte, len(members))
	for i, member := range members {
		entityAddrs[i] = member.EntityAddr
	}
	return entityAddrs
}

// Returns (nil, <firstUnknownEntityAddr>) if any unknown
func (gov *Governmint) loadEntities(entityAddrs [][]byte) ([]*types.Entity, []byte) {
	entities := make([]*types.Entity, len(entityAddrs))
	for i, entityAddr := range entityAddrs {
		entity, ok := gov.GetEntity(entityAddr)
		if !ok {
			return nil, entityAddr
		}
		entities[i] = entity
	}
	return entities, nil
}

func isMemberOf(group *types.Group, entityAddr []byte) bool {
	for _, member := range group.Members {
		if bytes.Equal(member.EntityAddr, entityAddr) {
			return true
		}
	}
	return false
}

func hasVoted(aProposal *types.ActiveProposal, entityAddr []byte) (bool, int) {
	for i, sVote := range aProposal.SignedVotes {
		if bytes.Equal(sVote.Vote.EntityAddr, entityAddr) {
			return true, i
		}
	}
	return false, -1
}

//----------------------------------------

// Get some object, or panic.
// objPtr: pointer to the object to populate, if value exists for key
// Use the return value, so nil can be returned for keys with no value.
func (gov *Governmint) getObject(key []byte, objPtr interface{}) interface{} {
	res := gov.eyesCli.GetSync(key)
	if res.IsErr() {
		panic("Error getting obj: " + res.Error())
	}
	if len(res.Data) == 0 {
		return nil // NOTE must use return value
	}
	err := wire.ReadBinaryBytes(res.Data, objPtr)
	if err != nil {
		panic("Error parsing obj: " + err.Error())
	}
	return objPtr
}

// Set some object, or panic
// If obj is a concrete type of an interface,
// remember to wrap in struct{MyInterface}{obj}.
func (gov *Governmint) setObject(key []byte, obj interface{}) {
	objBytes := wire.BinaryBytes(obj)
	res := gov.eyesCli.SetSync(key, objBytes)
	if res.IsErr() {
		panic("Error setting obj: " + res.Error())
	}
}

func (gov *Governmint) GetEntity(addr []byte) (entity *types.Entity, ok bool) {
	obj := gov.getObject(types.EntityKey(addr), &types.Entity{})
	if obj == nil {
		return nil, false
	} else {
		return obj.(*types.Entity), true
	}
}

func (gov *Governmint) SetEntity(o *types.Entity) {
	gov.setObject(types.EntityKey(o.Addr), *o)
}

func (gov *Governmint) GetGroup(id string) (group *types.Group, ok bool) {
	obj := gov.getObject(types.GroupKey(id), &types.Group{})
	if obj == nil {
		return nil, false
	} else {
		return obj.(*types.Group), true
	}
}

func (gov *Governmint) SetGroup(o *types.Group) {
	gov.setObject(types.GroupKey(o.ID), *o)
}

func (gov *Governmint) GetActiveProposal(id string) (ap *types.ActiveProposal, ok bool) {
	obj := gov.getObject(types.ActiveProposalKey(id), &types.ActiveProposal{})
	if obj == nil {
		return nil, false
	} else {
		return obj.(*types.ActiveProposal), true
	}
}

func (gov *Governmint) SetActiveProposal(o *types.ActiveProposal) {
	gov.setObject(types.ActiveProposalKey(o.Proposal.ID), *o)
}

func (gov *Governmint) GetGovMeta() (ap *types.GovMeta, ok bool) {
	obj := gov.getObject(types.GovMetaKey(), &types.GovMeta{})
	if obj == nil {
		return nil, false
	} else {
		return obj.(*types.GovMeta), true
	}
}

func (gov *Governmint) SetGovMeta(o *types.GovMeta) {
	gov.setObject(types.GovMetaKey(), *o)
}
