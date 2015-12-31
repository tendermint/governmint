package governmint

import (
	"sync"

	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-merkle"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/tmsp/types"
)

type GovernmintApplication struct {
	mtx sync.Mutex

	// contains state roots of trees for groups, members, proposals, resolutions
	state merkle.Tree
	/*
		groups      merkle.Tree
		members     merkle.Tree
		proposals   merkle.Tree
		resolutions merkle.Tree
	*/
}

func NewGovernmintApplication() *GovernmintApplication {
	state := merkle.NewIAVLTree(
		wire.BasicCodec,
		wire.BasicCodec,
		0,
		nil,
	)
	return &GovernmintApplication{state: state}
}

func (dapp *GovernmintApplication) Open() types.AppContext {
	dapp.mtx.Lock()
	defer dapp.mtx.Unlock()
	return &GovernmintAppContext{
		app: dapp,
		Governmint: &Governmint{
			state: dapp.state.Copy(),
		},
	}
}

func (dapp *GovernmintApplication) commitState(state merkle.Tree) {
	dapp.mtx.Lock()
	defer dapp.mtx.Unlock()
	dapp.state = state.Copy()
}

func (dapp *GovernmintApplication) getState() merkle.Tree {
	dapp.mtx.Lock()
	defer dapp.mtx.Unlock()
	return dapp.state.Copy()
}

//--------------------------------------------------------------------------------

type GovernmintAppContext struct {
	app *GovernmintApplication
	*Governmint
}

func (gov *GovernmintAppContext) Echo(message string) string {
	return message
}

func (gov *GovernmintAppContext) Info() []string {
	return []string{Fmt("")}
}

func (gov *GovernmintAppContext) SetOption(key string, value string) types.RetCode {
	return 0
}

func (gov *GovernmintAppContext) AppendTx(txBytes []byte) ([]types.Event, types.RetCode) {
	var tx SignedTx
	var err error
	wire.ReadJSON(&tx, txBytes, &err)
	if err != nil {
		return nil, types.RetCodeEncodingError
	}

	var retCode types.RetCode
	switch tx_ := tx.Tx.(type) {
	case *ProposalTx:
		retCode = gov.addProposal(tx_, tx.Signature)
	case *VoteTx:
		retCode = gov.addVote(tx_, tx.Signature)
	default:
		retCode = types.RetCodeUnknownRequest
	}

	return nil, retCode
}

func (gov *GovernmintAppContext) GetHash() ([]byte, types.RetCode) {
	// TODO  ...

	hash := gov.state.Hash()
	return hash, 0
}

func (gov *GovernmintAppContext) Commit() types.RetCode {
	// TODO ...
	gov.app.commitState(gov.state)
	return 0
}

func (gov *GovernmintAppContext) Rollback() types.RetCode {
	gov.state = gov.app.getState()
	return 0
}

func (gov *GovernmintAppContext) AddListener(key string) types.RetCode {
	return 0
}

func (gov *GovernmintAppContext) RemListener(key string) types.RetCode {
	return 0
}

func (gov *GovernmintAppContext) Close() error {
	return nil
}

//--------------------------------------------------------
