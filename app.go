package governmint

import (
	"sync"

	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/tmsp/types"
)

type GovernmintApplication struct {
	mtx sync.Mutex
	*Governmint
}

func NewGovernmintApplication(govFile string) *GovernmintApplication {
	gov, err := loadGovFromFile(govFile)
	if err != nil {
		Exit(err.Error())
	}
	return &GovernmintApplication{Governmint: gov}
}

func (govApp *GovernmintApplication) Open() types.AppContext {
	govApp.mtx.Lock()
	defer govApp.mtx.Unlock()
	return &GovernmintAppContext{
		app:        govApp,
		Governmint: govApp.Governmint.Copy(),
	}
}

func (govApp *GovernmintApplication) commitGov(gov *Governmint) {
	govApp.mtx.Lock()
	defer govApp.mtx.Unlock()
	govApp.Governmint = gov.Copy()
}

func (govApp *GovernmintApplication) getGov() *Governmint {
	govApp.mtx.Lock()
	defer govApp.mtx.Unlock()
	return govApp.Governmint.Copy()
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
	log.Warn(string(txBytes))
	wire.ReadJSON(&tx, txBytes, &err)
	if err != nil {
		log.Error("Encoding error", "error", err)
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

	log.Warn("RetCode", retCode)
	return nil, retCode
}

func (gov *GovernmintAppContext) GetHash() ([]byte, types.RetCode) {
	// TODO  ...

	hash := gov.state.Hash()
	return hash, 0
}

func (gov *GovernmintAppContext) Commit() types.RetCode {
	// TODO ...
	gov.app.commitGov(gov.Governmint)
	return 0
}

func (gov *GovernmintAppContext) Rollback() types.RetCode {
	gov.Governmint = gov.app.getGov()
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
