package gov

import (
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

// TMSP::Info
func (gov *Governmint) Info() string {
	return "Governmint v" + Version
}

func (gov *Governmint) RunTxParsed(tx types.Tx) (code tmsp.CodeType, result []byte, log string) {
	switch tx := tx.(type) {
	case *types.ProposalTx:
		return gov.RunProposalTx(tx)
	case *types.VoteTx:
		return gov.RunVoteTx(tx)
	default:
		PanicSanity("Unknown tx type")
		return
	}
}

func (gov *Governmint) RunProposalTx(tx *types.ProposalTx) (code tmsp.CodeType, result []byte, log string) {
	// Ensure that proposer exists
	entity, ok := gov.GetEntity(tx.EntityID)
	if !ok {
		return tmsp.CodeType_GovUnknownEntity, nil, Fmt("Entity %v unknown", tx.EntityID)
	}
	// Ensure signature is valid
	signBytes := tx.SignBytes()
	if !entity.PubKey.VerifyBytes(signBytes, tx.Signature) {
		return tmsp.CodeType_Unauthorized, nil, Fmt("Invalid signature")
	}
	// Ensure that the proposal is valid
	tmspErr := gov.validateProposal(tx.Proposal, entity)
	if !tmspErr.IsOK() {
		return tmspErr.Code, nil, tmspErr.Log
	}
	// Good! Create a new proposal
	proposal := tx.Proposal
	aProposal := &types.ActiveProposal{
		Proposal:    proposal,
		SignedVotes: nil,
	}
	gov.SetActiveProposal(aProposal)
	return tmsp.CodeType_OK, nil, Fmt("Proposal created")
}

func (gov *Governmint) RunVoteTx(tx *types.VoteTx) (code tmsp.CodeType, result []byte, log string) {
	// Ensure that voter exists
	entity, ok := gov.GetEntity(tx.Vote.EntityID)
	if !ok {
		return tmsp.CodeType_GovUnknownEntity, nil, Fmt("Entity %v unknown", tx.Vote.EntityID)
	}
	// Ensure signature is valid
	signBytes := tx.SignBytes()
	if !entity.PubKey.VerifyBytes(signBytes, tx.Signature) {
		return tmsp.CodeType_Unauthorized, nil, Fmt("Invalid signature")
	}
	// Ensure that the proposal exists
	aProposal, ok := gov.GetActiveProposal(tx.Vote.ProposalID)
	if !ok {
		return tmsp.CodeType_GovUnknownProposal, nil, Fmt("Unknown proposal %v", tx.Vote.ProposalID)
	}
	// Ensure that the vote's height is <= current height
	if !(tx.Vote.Height <= gov.GovMeta.Height) {
		return tmsp.CodeType_GovInvalidVote, nil, Fmt("Vote height is invalid")
	}
	// Ensure that the vote's height matches the proposal's range
	if !(aProposal.StartHeight <= tx.Vote.Height &&
		tx.Vote.Height <= aProposal.EndHeight) {
		return tmsp.CodeType_GovInvalidVote, nil, Fmt("Vote height is invalid")
	}
	// Ensure that the group exists, if specified
	var group *types.Group
	// XXX refactor
	groupID := tx.Proposal.GetGroupID()
	if groupID != "" {
		ok := false
		group, ok := gov.GetGroup(groupID)
		if !ok {
			return tmsp.CodeType_GovUnknownGroup, nil, Fmt("Group %v unknown", groupID)
		}
	}
	// Ensure that the voter belongs to the group
	if !isMemberOf(group, entity.ID) {
		return tmsp.CodeType_GovInvalidMember, nil, Fmt("Voter %v not a member of %v", entity.ID, groupID)
	}
	// Ensure that the voter hasn't already voted
	if exists, _ := hasVoted(aProposal, entity.ID); exists {
		return tmsp.CodeType_GovDuplicateVote, nil, Fmt("Voter %v already voted", entity.ID)
	}
	// Good! Add a SignedVote
	aProposal.SignedVotes = append(aProposal.SignedVotes, types.SignedVote{
		Vote:      tx.Vote,
		Signature: tx.Signature,
	})
	gov.SetActiveProposal(aProposal)
	return tmsp.CodeType_OK, nil, Fmt("Vote added to ActiveProposal")
}

/*
// TMSP::CheckTx
func (gov *Governmint) CheckTx(txBytes []byte) (code tmsp.CodeType, result []byte, log string) {
	return // XXX
}
*/

/*
// TMSP::Query
func (gov *Governmint) Query(query []byte) (code tmsp.CodeType, result []byte, log string) {
	return // XXX
}
*/

// TMSP::InitChain
func (gov *Governmint) InitChain(validators []*tmsp.Validator) {
	// Construct a group of entities for the validators.
	vGroup := &types.Group{
		ID:      types.ValidatorsGroupID,
		Version: 0,
	}
	for i, validator := range validators {
		var pubKey crypto.PubKey
		err := wire.ReadBinaryBytes(validator.PubKey, &pubKey)
		if err != nil {
			PanicSanity("Error reading validator pubkey: " + err.Error())
		}
		// Create an entity with this validator
		entity := &types.Entity{
			ID:     Fmt("%v", i),
			PubKey: pubKey,
		}
		gov.SetEntity(entity)
		// Add as member
		member := types.Member{
			EntityID:    entity.ID,
			VotingPower: validator.Power,
		}
		vGroup.Members = append(vGroup.Members, member)
	}
	// Save vGroup
	gov.SetGroup(vGroup)
}

// TMSP::EndBlock
func (gov *Governmint) EndBlock(height uint64) (changedValidators []*tmsp.Validator) {
	gov.GovMeta.Height = height + 1
	// Persist GovMeta
	gov.SetGovMeta(gov.GovMeta)
	// XXX Changed Validators
	return
}

//----------------------------------------

func (gov *Governmint) validateProposal(p types.Proposal, proposer *types.Entity) (err types.TMSPError) {
	// Ensure that the proposal is unique
	if _, exists := gov.GetActiveProposal(p.ID); exists {
		return types.TMSPError{tmsp.CodeType_GovDuplicateProposal,
			Fmt("Proposal with id %v already exists", p.ID)}
	}
	// Type dependent checks
	switch p := p.Info.(type) {
	case *types.GroupCreateProposalInfo:
		// Ensure that the group does not exist
		if _, exists := gov.GetGroup(p.GroupID); exists {
			return types.TMSPError{tmsp.CodeType_GovDuplicateGroup,
				Fmt("Group with id %v already exists", p.GroupID)}
		}
		// Ensure that the member entities are unique
		if ok, dupe := validateUniqueMembers(p.Members); !ok {
			return types.TMSPError{tmsp.CodeType_GovDuplicateMember,
				Fmt("Duplicate member %v", dupe)}
		}
		// Ensure that the member voting powers are reasonable
		for _, member := range p.Members {
			if member.VotingPower == 0 {
				return types.TMSPError{tmsp.CodeType_GovInvalidVotingPower,
					Fmt("Member cannot have 0 voting power")}
			}
			if member.VotingPower > MaxVotingPower {
				return types.TMSPError{tmsp.CodeType_GovInvalidVotingPower,
					Fmt("Member voting power too large")}
			}
		}
		// Ensure that all the entities exist
		entityIDs := entityIDsFromMembers(p.Members)
		entities, unknownEntityID := gov.loadEntities(entityIDs)
		if unknownEntityID != "" {
			return types.TMSPError{tmsp.CodeType_GovUnknownEntity,
				Fmt("Group creation with unknown entity %v", unknownEntityID)}
		}
	case *types.GroupUpdateProposalInfo:
		// Ensure that the group exists
		group, ok := gov.GetGroup(p.GroupID)
		if !ok {
			return types.TMSPError{tmsp.CodeType_GovUnknownGroup,
				Fmt("Group with id %v doesn't exist", p.GroupID)}
		}
		// Ensure that the proposer belongs to the group
		if !isMemberOf(group, proposer.ID) {
			return types.TMSPError{tmsp.CodeType_Unauthorized,
				Fmt("Proposer %v is not member of %v", proposer.ID, group.ID)}
		}
		// Ensure that the member entities are unique
		if ok, dupe := validateUniqueMembers(p.ChangedMembers); !ok {
			return types.TMSPError{tmsp.CodeType_GovDuplicateMember,
				Fmt("Duplicate member %v", dupe)}
		}
		// Ensure that the member voting powers are reasonable
		for _, member := range p.ChangedMembers {
			if member.VotingPower == 0 {
				// This is fine, we're removing members.
			}
			if member.VotingPower > MaxVotingPower {
				return types.TMSPError{tmsp.CodeType_GovInvalidVotingPower,
					Fmt("Member voting power too large")}
			}
		}
		// Ensure that all the entities exist
		entityIDs := entityIDsFromMembers(p.ChangedMembers)
		entities, unknownEntityID := gov.loadEntities(entityIDs)
		if unknownEntityID != "" {
			return types.TMSPError{tmsp.CodeType_GovUnknownEntity,
				Fmt("Group creation with unknown entity %v", unknownEntityID)}
		}
	case *types.TextProposalInfo:
		// Ensure that the group exists
		group, ok := gov.GetGroup(p.GroupID)
		if !ok {
			return types.TMSPError{tmsp.CodeType_GovUnknownGroup,
				Fmt("Group with id %v doesn't exist", p.GroupID)}
		}
		// Ensure that the proposer belongs to the group
		if !isMemberOf(group, proposer.ID) {
			return types.TMSPError{tmsp.CodeType_Unauthorized,
				Fmt("Proposer %v is not member of %v", proposer.ID, group.ID)}
		}
	case *types.UpgradeProposalInfo:
		// Ensure that the Admin group exists
		group, ok := gov.GetGroup(types.AdminGroupID)
		if !ok {
			return types.TMSPError{tmsp.CodeType_GovUnknownGroup,
				Fmt("Admin group does not exist")}
		}
		// Ensure that the proposer belongs to the group
		if !isMemberOf(group, proposer.ID) {
			return types.TMSPError{tmsp.CodeType_Unauthorized,
				Fmt("Proposer %v is not member of %v", proposer.ID, group.ID)}
		}
		// Ensure that the number of modules is > 0.
		if len(p.Modules) == 0 {
			return types.TMSPError{tmsp.CodeType_EncodingError,
				Fmt("Software upgrade requires > 0 modules")}
		}
	}
	return types.TMSPError{tmsp.CodeType_OK, ""}
}

// Returns (true, "") if members are unique
// Returns (false, <duplicateEntityID>) if members are not unique
// NOTE: zero members is fine.
func validateUniqueMembers(members []types.Member) (bool, string) {
	entityIDs := map[string]struct{}{}
	for _, member := range members {
		if _, exists := entityIDs[member.EntityID]; exists {
			return false, member.EntityID
		}
		entityIDs[member.EntityID] = struct{}{}
	}
	return true, ""
}

func entityIDsFromMembers(members []types.Member) []string {
	entityIDs := make([]string, len(members))
	for i, member := range members {
		entityIDs[i] = member.EntityID
	}
	return entityIDs
}

// Returns (nil, <firstUnknownEntityID>) if any unknown
func (gov *Governmint) loadEntities(entityIDs []string) ([]*types.Entity, string) {
	entities := make([]*types.Entity, len(entityIDs))
	for i, entityID := range entityIDs {
		entity, ok := gov.GetEntity(entityID)
		if !ok {
			return nil, entityID
		}
		entities[i] = entity
	}
	return entities, ""
}

func isMemberOf(group *types.Group, entityID string) bool {
	for _, member := range group.Members {
		if member.EntityID == entityID {
			return true
		}
	}
	return false
}

func hasVoted(aProposal *types.ActiveProposal, entityID string) (bool, int) {
	for i, sVote := range aProposal.SignedVotes {
		if sVote.Vote.EntityID == entityID {
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
	objBytes, err := gov.eyesCli.GetSync(key)
	if err != nil {
		panic("Error getting obj: " + err.Error())
	}
	if len(objBytes) == 0 {
		return nil // NOTE must use return value
	}
	err = wire.ReadBinaryBytes(objBytes, objPtr)
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
	err := gov.eyesCli.SetSync(key, objBytes)
	if err != nil {
		panic("Error setting obj: " + err.Error())
	}
}

func (gov *Governmint) GetEntity(id string) (entity *types.Entity, ok bool) {
	obj := gov.getObject(types.EntityKey(id), &types.Entity{})
	if obj == nil {
		return nil, false
	} else {
		return obj.(*types.Entity), true
	}
}

func (gov *Governmint) SetEntity(o *types.Entity) {
	gov.setObject(types.EntityKey(o.ID), *o)
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
