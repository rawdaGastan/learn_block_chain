package node

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rawdaGastan/learn_block_chain/internal"
)

type TxAddReq struct {
	From     string `json:"from"`
	FromPwd  string `json:"from_pwd"`
	To       string `json:"to"`
	Gas      uint   `json:"gas"`
	GasPrice uint   `json:"gasPrice"`
	Value    uint   `json:"value"`
	Data     string `json:"data"`
}

type ErrRes struct {
	Error string `json:"error"`
}

type BalancesRes struct {
	Hash     internal.Hash             `json:"block_hash"`
	Balances map[internal.Account]uint `json:"balances"`
}

type TxAddRes struct {
	Success bool `json:"success"`
}

type StatusRes struct {
	Hash       internal.Hash       `json:"block_hash"`
	Number     uint64              `json:"block_number"`
	KnownPeers map[string]PeerNode `json:"peers_known"`
	PendingTXs []internal.Tx       `json:"pending_txs"`
}

type SyncRes struct {
	Blocks []internal.Block `json:"blocks"`
}

type AddPeerRes struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func writeErrRes(w http.ResponseWriter, err error) {
	jsonErrRes, _ := json.Marshal(ErrRes{err.Error()})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(jsonErrRes)
}

func writeRes(w http.ResponseWriter, content interface{}) {
	contentJson, err := json.Marshal(content)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(contentJson)
}

func readReq(r *http.Request, reqBody interface{}) error {
	reqBodyJson, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read request body. %s", err.Error())
	}
	defer r.Body.Close()

	err = json.Unmarshal(reqBodyJson, reqBody)
	if err != nil {
		return fmt.Errorf("unable to unmarshal request body. %s", err.Error())
	}

	return nil
}

func readRes(r *http.Response, reqBody interface{}) error {
	reqBodyJson, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read response body. %s", err.Error())
	}
	defer r.Body.Close()

	err = json.Unmarshal(reqBodyJson, reqBody)
	if err != nil {
		return fmt.Errorf("unable to unmarshal response body. %s", err.Error())
	}

	return nil
}
