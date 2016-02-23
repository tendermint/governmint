package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/governmint/types"
)

var KeyHost = "http://localhost:4767"

func SignTx(tx types.Tx, keyName string) (crypto.Signature, error) {
	buf := new(bytes.Buffer)
	var n int
	var err error
	wire.WriteJSON(tx, buf, &n, &err)
	if err != nil {
		return nil, err
	}

	args := map[string]string{
		"msg":  hex.EncodeToString(buf.Bytes()),
		"name": keyName,
	}
	sigS, err := RequestResponse(KeyHost, "sign", args)
	if err != nil {
		return nil, err
	}
	sigBytes, err := hex.DecodeString(sigS)
	if err != nil {
		return nil, err
	}
	var sig crypto.SignatureEd25519
	copy(sig[:], sigBytes)
	return sig, nil
}

type HTTPResponse struct {
	Response string
	Error    string
}

func RequestResponse(addr, method string, args map[string]string) (string, error) {
	b, err := json.Marshal(args)
	if err != nil {
		return "", err
	}
	endpoint := fmt.Sprintf("%s/%s", addr, method)
	log.Debug(Fmt("Sending request body (%s): %s\n", endpoint, string(b)))
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	res, errS, err := requestResponse(req)
	if err != nil {
		return "", fmt.Errorf("Error calling eris-keys at %s: %s", endpoint, err.Error())
	}
	if errS != "" {
		return "", fmt.Errorf("Error (string) calling eris-keys at %s: %s", endpoint, errS)
	}
	return res, nil
}

func requestResponse(req *http.Request) (string, string, error) {
	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf(resp.Status)
	}
	return unpackResponse(resp)
}

func unpackResponse(resp *http.Response) (string, string, error) {
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	r := new(HTTPResponse)
	if err := json.Unmarshal(b, r); err != nil {
		return "", "", err
	}
	return r.Response, r.Error, nil
}
