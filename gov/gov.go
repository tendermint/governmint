package gov

import (
	"github.com/tendermint/go-wire"
	"github.com/tendermint/governmint/types"
	eyes "github.com/tendermint/merkleeyes/client"
	tmsp "github.com/tendermint/tmsp/types"
)

type Governmint struct {
	eyesCli *eyes.Client
}

func NewGovernmint(eyesCli *eyes.Client) *Governmint {
	return &Governmint{eyesCli}
}

func (gov *Governmint) AppendTx(txBytes []byte) (code tmsp.CodeType, result []byte, log string) {
	var tx types.Tx
	err := wire.ReadBinaryBytes(txBytes, &tx)
	if err != nil {
		return tmsp.CodeType_EncodingError, nil, "Error parsing tx bytes: " + err.Error()
	}

	// Get entity

	// Verify signature
	//signBytes := tx.SignBytes()

	return // XXX
}

func (gov *Governmint) CheckTx(txBytes []byte) (code tmsp.CodeType, result []byte, log string) {
	return // XXX
}

func (gov *Governmint) Query(query []byte) (code tmsp.CodeType, result []byte, log string) {
	return // XXX
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

func (gov *Governmint) getEntity(id string) (entity *types.Entity, ok bool) {
	obj := gov.getObject(types.EntityKey(id), &types.Entity{})
	if obj == nil {
		return nil, false
	} else {
		return obj.(*types.Entity), true
	}
}

func (gov *Governmint) setEntity(o *types.Entity) {
	gov.setObject(types.EntityKey(o.ID), *o)
}

func (gov *Governmint) getGroup(id string) (group *types.Group, ok bool) {
	obj := gov.getObject(types.GroupKey(id), &types.Group{})
	if obj == nil {
		return nil, false
	} else {
		return obj.(*types.Group), true
	}
}

func (gov *Governmint) setGroup(o *types.Group) {
	gov.setObject(types.GroupKey(o.ID), *o)
}

func (gov *Governmint) getActiveProposal(id string) (ap *types.ActiveProposal, ok bool) {
	obj := gov.getObject(types.ActiveProposalKey(id), &types.ActiveProposal{})
	if obj == nil {
		return nil, false
	} else {
		return obj.(*types.ActiveProposal), true
	}
}

func (gov *Governmint) setActiveProposal(o *types.ActiveProposal) {
	gov.setObject(types.ActiveProposalKey(types.ProposalID(o)), *o)
}
