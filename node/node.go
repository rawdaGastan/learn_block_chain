package node

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rawdaGastan/learn_block_chain/internal"
)

const DefaultHTTPort = 8080

type PeerNode struct {
	IP          string `json:"ip"`
	Port        uint64 `json:"port"`
	IsBootstrap bool   `json:"is_bootstrap"`

	// Whenever my node already established connection, sync with this Peer
	connected bool
}

type Node struct {
	dataDir    string
	port       uint64 // To inject the State into HTTP handlers
	state      *internal.State
	knownPeers []PeerNode
}

func New(dataDir string, port uint64, bootstrap PeerNode) *Node {
	return &Node{
		dataDir:    dataDir,
		port:       port,
		knownPeers: []PeerNode{bootstrap},
	}
}

func (n *Node) Run() error {
	fmt.Println(fmt.Sprintf("Listening on HTTP port: %d", n.port))
	state, err := internal.NewStateFromDisk(n.dataDir)
	if err != nil {
		return err
	}
	defer state.Close()
	n.state = state
	http.HandleFunc("/balances/list", func(w http.ResponseWriter, r *http.Request) {
		listBalancesHandler(w, r, state)
	})
	http.HandleFunc("/tx/add", func(w http.ResponseWriter, r *http.Request) {
		txAddHandler(w, r, state)
	})
	http.HandleFunc("/node/status", func(w http.ResponseWriter, r *http.Request) {
		statusHandler(w, r, n)
	})
	return http.ListenAndServe(fmt.Sprintf(":%d", n.port), nil)
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
