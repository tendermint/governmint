package gov

import (
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/governmint/types"
	eyesApp "github.com/tendermint/merkleeyes/app"
	eyes "github.com/tendermint/merkleeyes/client"
	"github.com/tendermint/tmsp/server"
	"testing"
)

func makeMerkleEyesServer(addr string) *server.Server {
	app := eyesApp.NewMerkleEyesApp()
	s, err := server.NewServer(addr, app)
	if err != nil {
		panic("starting MerkleEyes listener: " + err.Error())
	}
	return s
}

func makeMerkleEyesClient(addr string) *eyes.Client {
	c, err := eyes.NewClient("unix://test.sock")
	if err != nil {
		panic("creating MerkleEyes client: " + err.Error())
	}
	return c
}

func TestUnit(t *testing.T) {
	s := makeMerkleEyesServer("unix://test.sock")
	defer s.Stop()

	c := makeMerkleEyesClient("unix://test.sock")
	defer c.Stop()

	privKey := crypto.GenPrivKeyEd25519()
	pubKey := privKey.PubKey()

	gov := NewGovernmint(c)
	gov.setEntity(&types.Entity{
		ID:     "my_entity_id",
		PubKey: pubKey,
	})

	entityCopy, ok := gov.getEntity("my_entity_id")
	if !ok {
		t.Error("Saved(set) entity does not exist")
	}
	if entityCopy.ID != "my_entity_id" {
		t.Error("Got wrong entity id")
	}
	if !pubKey.Equals(entityCopy.PubKey) {
		t.Error("Got wrong entity pubkey")
	}

	entityBad, ok := gov.getEntity("my_bad_id")
	if ok || entityBad != nil {
		t.Error("Expected nil entity")
	}
}
