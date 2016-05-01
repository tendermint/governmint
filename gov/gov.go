package gov

import (
	"bytes"
	"fmt"
	"github.com/tendermint/go-common"

	base "github.com/tendermint/basecoin/types"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/governmint/types"
	tmsp "github.com/tendermint/tmsp/types"
)

const (
	Version        = "0.1"
	MaxVotingPower = 1<<53 - 1
)

type Governmint struct {
	*types.GovMeta
}

func NewGovernmint() *Governmint {
	gov := &Governmint{
		GovMeta: &types.GovMeta{
			Height: 0,
		},
	}
	return gov
}

func (gov *Governmint) SetOption(store base.KVStore, key string, value string) (log string) {
	switch key {
	case "admin":
		// Read entity
		var entity = new(types.Entity)
		err := wire.ReadJSONBytes([]byte(value), entity)
		if err != nil {
			return "Error decoding admin entity: " + err.Error()
		}
		// Save entity
		gov.SetEntity(store, entity)
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
		gov.SetGroup(store, adminGroup)
		return "Success"
	case "entity":
		// Read entity
		var entity = new(types.Entity)
		err := wire.ReadJSONBytes([]byte(value), entity)
		if err != nil {
			return "Error decoding entity: " + err.Error()
		}
		// Save entity
		gov.SetEntity(store, entity)
	}
	return "Unrecognized governmint option key " + key
}

// Implements basecoin.Plugin
func (gov *Governmint) RunTx(store base.KVStore, ctx base.CallContext, txBytes []byte) tmsp.Result {
	var tx types.Tx
	err := wire.ReadBinaryBytes(txBytes, &tx)
	if err != nil {
		return tmsp.ErrEncodingError.SetLog(
			Fmt("Error parsing Governmint tx bytes: %v", err.Error()))
	}
	return gov.RunTxParsed(store, tx)
}

func (gov *Governmint) RunTxParsed(store base.KVStore, tx types.Tx) tmsp.Result {
	switch tx := tx.(type) {
	case *types.ProposalTx:
		return gov.RunProposalTx(store, tx)
	case *types.VoteTx:
		return gov.RunVoteTx(store, tx)
	default:
		PanicSanity("Unknown tx type")
		return tmsp.NewError(tmsp.CodeType_InternalError, "Unknown tx type")
	}
}

func (gov *Governmint) RunProposalTx(store base.KVStore, tx *types.ProposalTx) tmsp.Result {
	// Ensure that proposer exists
	entity, ok := gov.GetEntity(store, tx.EntityAddr)
	if !ok {
		return tmsp.NewError(tmsp.CodeType_GovUnknownEntity,
			Fmt("Entity %X unknown", tx.EntityAddr))
	}
	// Ensure signature is valid
	signBytes := tx.SignBytes()
	if !entity.PubKey.VerifyBytes(signBytes, tx.Signature) {
		return tmsp.NewError(tmsp.CodeType_Unauthorized,
			Fmt("Invalid signature"))
	}
	// Ensure that the proposal is valid
	tmspErr := gov.validateProposal(store, tx.Proposal, entity)
	if !tmspErr.IsOK() {
		return tmspErr
	}
	// Good! Create a new proposal
	proposal := tx.Proposal
	aProposal := &types.ActiveProposal{
		Proposal:    proposal,
		SignedVotes: nil,
	}
	gov.SetActiveProposal(store, aProposal)
	return tmsp.NewResultOK(nil, "Proposal created")
}

func (gov *Governmint) RunVoteTx(store base.KVStore, tx *types.VoteTx) tmsp.Result {
	// Ensure that voter exists
	entity, ok := gov.GetEntity(store, tx.Vote.EntityAddr)
	if !ok {
		return tmsp.NewError(tmsp.CodeType_GovUnknownEntity,
			Fmt("Entity %X unknown", tx.Vote.EntityAddr))
	}
	// Ensure signature is valid
	signBytes := tx.SignBytes()
	if !entity.PubKey.VerifyBytes(signBytes, tx.Signature) {
		return tmsp.NewError(tmsp.CodeType_Unauthorized,
			Fmt("Invalid signature"))
	}
	// Ensure that the proposal exists
	aProposal, ok := gov.GetActiveProposal(store, tx.Vote.ProposalID)
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
	voteGroup, ok := gov.GetGroup(store, aProposal.VoteGroupID)
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
	gov.SetActiveProposal(store, aProposal)
	return tmsp.NewResultOK(nil, "Vote added to ActiveProposal")
}

func (gov *Governmint) InitChain(store base.KVStore, validators []*tmsp.Validator) {
	fmt.Println(common.Red(Fmt(">> B")))
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
		gov.SetEntity(store, entity)
		// Add as member
		member := types.Member{
			EntityAddr:  entity.Addr,
			VotingPower: validator.Power,
		}
		vGroup.Members = append(vGroup.Members, member)
	}
	// Save vGroup
	fmt.Println(common.Red(Fmt(">> %v", vGroup)))
	gov.SetGroup(store, vGroup)
}

func (gov *Governmint) BeginBlock(store base.KVStore, height uint64) {
	if govMeta, ok := gov.GetGovMeta(store); ok {
		gov.GovMeta = govMeta
	}
	gov.GovMeta.Height = height
	return
}

func (gov *Governmint) EndBlock(store base.KVStore, height uint64) []*tmsp.Validator {
	gov.SetGovMeta(store, gov.GovMeta)
	return nil // XXX Return changes to validator set
}

//----------------------------------------

func (gov *Governmint) validateProposal(store base.KVStore, p types.Proposal, proposer *types.Entity) (res tmsp.Result) {
	// Ensure that the proposal is unique
	if _, exists := gov.GetActiveProposal(store, p.ID); exists {
		return tmsp.NewError(tmsp.CodeType_GovDuplicateProposal,
			Fmt("Proposal with id %v already exists", p.ID))
	}
	// Ensure that the voting group exists
	voteGroup, ok := gov.GetGroup(store, p.VoteGroupID)
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
		if _, exists := gov.GetGroup(store, pInfo.NewGroupID); exists {
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
		_, unknownEntityAddr := gov.loadEntities(store, entityAddrs)
		if unknownEntityAddr != nil {
			return tmsp.NewError(tmsp.CodeType_GovUnknownEntity,
				Fmt("Group creation with unknown entity %X", unknownEntityAddr))
		}
	case *types.GroupUpdateProposalInfo:
		// Ensure that the update group exists
		updateGroup, ok := gov.GetGroup(store, pInfo.UpdateGroupID)
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
		_, unknownEntityAddr := gov.loadEntities(store, entityAddrs)
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
func (gov *Governmint) loadEntities(store base.KVStore, entityAddrs [][]byte) ([]*types.Entity, []byte) {
	entities := make([]*types.Entity, len(entityAddrs))
	for i, entityAddr := range entityAddrs {
		entity, ok := gov.GetEntity(store, entityAddr)
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
func (gov *Governmint) getObject(store base.KVStore, key []byte, objPtr interface{}) interface{} {
	valueBytes := store.Get(key)
	if len(valueBytes) == 0 {
		return nil // NOTE must use return value
	}
	err := wire.ReadBinaryBytes(valueBytes, objPtr)
	if err != nil {
		panic("Error parsing obj: " + err.Error())
	}
	return objPtr
}

// Set some object, or panic
// If obj is a concrete type of an interface,
// remember to wrap in struct{MyInterface}{obj}.
func (gov *Governmint) setObject(store base.KVStore, key []byte, obj interface{}) {
	objBytes := wire.BinaryBytes(obj)
	store.Set(key, objBytes)
}

func (gov *Governmint) GetEntity(store base.KVStore, addr []byte) (entity *types.Entity, ok bool) {
	obj := gov.getObject(store, types.EntityKey(addr), &types.Entity{})
	if obj == nil {
		return nil, false
	} else {
		return obj.(*types.Entity), true
	}
}

func (gov *Governmint) SetEntity(store base.KVStore, o *types.Entity) {
	gov.setObject(store, types.EntityKey(o.Addr), *o)
}

func (gov *Governmint) GetGroup(store base.KVStore, id string) (group *types.Group, ok bool) {
	obj := gov.getObject(store, types.GroupKey(id), &types.Group{})
	if obj == nil {
		return nil, false
	} else {
		return obj.(*types.Group), true
	}
}

func (gov *Governmint) SetGroup(store base.KVStore, o *types.Group) {
	gov.setObject(store, types.GroupKey(o.ID), *o)
}

func (gov *Governmint) GetActiveProposal(store base.KVStore, id string) (ap *types.ActiveProposal, ok bool) {
	obj := gov.getObject(store, types.ActiveProposalKey(id), &types.ActiveProposal{})
	if obj == nil {
		return nil, false
	} else {
		return obj.(*types.ActiveProposal), true
	}
}

func (gov *Governmint) SetActiveProposal(store base.KVStore, o *types.ActiveProposal) {
	gov.setObject(store, types.ActiveProposalKey(o.Proposal.ID), *o)
}

func (gov *Governmint) GetGovMeta(store base.KVStore) (ap *types.GovMeta, ok bool) {
	obj := gov.getObject(store, types.GovMetaKey(), &types.GovMeta{})
	if obj == nil {
		return nil, false
	} else {
		return obj.(*types.GovMeta), true
	}
}

func (gov *Governmint) SetGovMeta(store base.KVStore, o *types.GovMeta) {
	gov.setObject(store, types.GovMetaKey(), *o)
}
