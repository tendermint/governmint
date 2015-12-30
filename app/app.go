package governmint

import (
	"bytes"
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
		app:   dapp,
		state: dapp.state.Copy(),
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
	app   *GovernmintApplication
	state merkle.Tree

	members   map[string]*Member
	groups    map[string]*Group
	proposals map[string]*Proposal
}

func (dac *GovernmintAppContext) Echo(message string) string {
	return message
}

func (dac *GovernmintAppContext) Info() []string {
	return []string{Fmt("members:%d, groups:%d, proposals:%d", len(dac.members), len(dac.groups), len(dac.proposals))}
}

func (dac *GovernmintAppContext) SetOption(key string, value string) types.RetCode {
	return 0
}

func (dac *GovernmintAppContext) AppendTx(txBytes []byte) ([]types.Event, types.RetCode) {
	var tx Tx
	var n int
	var err error
	wire.ReadBinary(&tx, bytes.NewBuffer(txBytes), 0, &n, &err)
	if err != nil {
		return nil, types.RetCodeEncodingError
	}
	switch tx_ := tx.(type) {
	case *ProposalTx:
		err = dac.addProposal(tx_)
	case *VoteTx:
		err = dac.addVote(tx_)
	default:
		// TODO: invite and group txs?
		return nil, types.RetCodeUnknownRequest
	}

	if err != nil {
		return nil, types.RetCodeInternalError
	}
	return nil, 0
}

func (dac *GovernmintAppContext) GetHash() ([]byte, types.RetCode) {
	// TODO  ...

	hash := dac.state.Hash()
	return hash, 0
}

func (dac *GovernmintAppContext) Commit() types.RetCode {
	// TODO ...
	dac.app.commitState(dac.state)
	return 0
}

func (dac *GovernmintAppContext) Rollback() types.RetCode {
	dac.state = dac.app.getState()
	return 0
}

func (dac *GovernmintAppContext) AddListener(key string) types.RetCode {
	return 0
}

func (dac *GovernmintAppContext) RemListener(key string) types.RetCode {
	return 0
}

func (dac *GovernmintAppContext) Close() error {
	return nil
}

//--------------------------------------------------------

func (dac *GovernmintAppContext) addProposal(tx *ProposalTx) error {
	return nil
}

func (dac *GovernmintAppContext) addVote(tx *VoteTx) error {
	return nil
}
