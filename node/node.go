package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rawdaGastan/learn_block_chain/internal"
)

const miningIntervalSeconds = 10
const DefaultMiner = "0x0000000000000000000000000000000000000000"

type PeerNode struct {
	IP          string `json:"ip"`
	Port        uint64 `json:"port"`
	IsBootstrap bool   `json:"is_bootstrap"`

	// Whenever my node already established connection, sync with this Peer
	connected bool
	Account   common.Address
}

type Node struct {
	dataDir string
	info    PeerNode

	state           *internal.State
	knownPeers      map[string]PeerNode
	pendingTXs      map[string]internal.SignedTx
	archivedTXs     map[string]internal.SignedTx
	newSyncedBlocks chan internal.Block
	newPendingTXs   chan internal.SignedTx
	isMining        bool
}

func New(dataDir string, ip string, port uint64, acc common.Address, bootstrap PeerNode) *Node {
	knownPeers := make(map[string]PeerNode)
	knownPeers[bootstrap.TcpAddress()] = bootstrap

	return &Node{
		dataDir:         dataDir,
		info:            NewPeerNode(ip, port, false, true, acc),
		knownPeers:      knownPeers,
		pendingTXs:      make(map[string]internal.SignedTx),
		archivedTXs:     make(map[string]internal.SignedTx),
		newSyncedBlocks: make(chan internal.Block),
		newPendingTXs:   make(chan internal.SignedTx, 10000),
		isMining:        false,
	}
}

func (pn PeerNode) TcpAddress() string {
	return fmt.Sprintf("%s:%d", pn.IP, pn.Port)
}

func NewPeerNode(ip string, port uint64, isBootstrap bool, connected bool, miner common.Address) PeerNode {
	return PeerNode{ip, port, isBootstrap, connected, miner}
}

func (n *Node) Run(ctx context.Context) error {
	fmt.Printf("Listening on: %s:%d\n", n.info.IP, n.info.Port)

	state, err := internal.NewStateFromDisk(n.dataDir)
	if err != nil {
		return err
	}
	defer state.Close()
	n.state = state

	// Run sync() in a separate thread
	go n.sync(ctx)
	go n.mine(ctx)

	http.HandleFunc("/balances/list", func(w http.ResponseWriter, r *http.Request) {
		listBalancesHandler(w, r, state)
	})
	http.HandleFunc("/tx/add", func(w http.ResponseWriter, r *http.Request) {
		txAddHandler(w, r, n)
	})
	http.HandleFunc("/node/status", func(w http.ResponseWriter, r *http.Request) {
		statusHandler(w, r, n)
	})
	http.HandleFunc("/node/sync", func(w http.ResponseWriter, r *http.Request) {
		syncHandler(w, r, n)
	})
	http.HandleFunc("/node/peer", func(w http.ResponseWriter, r *http.Request) {
		addPeerHandler(w, r, n)
	})

	server := &http.Server{Addr: fmt.Sprintf(":%d", n.info.Port)}

	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()

	err = server.ListenAndServe()
	if err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (n *Node) mine(ctx context.Context) error {
	var miningCtx context.Context
	var stopCurrentMining context.CancelFunc

	ticker := time.NewTicker(time.Second * miningIntervalSeconds)

	for {
		select {
		case <-ticker.C:
			go func() {
				if len(n.pendingTXs) > 0 && !n.isMining {
					n.isMining = true

					miningCtx, stopCurrentMining = context.WithCancel(ctx)
					err := n.minePendingTXs(miningCtx)
					if err != nil {
						fmt.Printf("ERROR: %s\n", err)
					}

					n.isMining = false
				}
			}()

		case block := <-n.newSyncedBlocks:
			if n.isMining {
				blockHash, _ := block.Hash()
				fmt.Printf("\nPeer mined next Block '%s' faster :(\n", blockHash.Hex())

				n.removeMinedPendingTXs(block)
				stopCurrentMining()
			}

		case <-ctx.Done():
			ticker.Stop()
			return nil
		}
	}
}

func (n *Node) minePendingTXs(ctx context.Context) error {
	blockToMine := NewPendingBlock(
		n.state.LatestBlockHash(),
		n.state.NextBlockNumber(),
		n.info.Account,
		n.getPendingTXsAsArray(),
	)

	minedBlock, err := Mine(ctx, blockToMine)
	if err != nil {
		return err
	}

	n.removeMinedPendingTXs(minedBlock)

	_, err = n.state.AddBlock(minedBlock)
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) AddPendingTX(tx internal.SignedTx, fromPeer PeerNode) error {
	txHash, err := tx.Hash()
	if err != nil {
		return err
	}

	txJson, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	_, isAlreadyPending := n.pendingTXs[txHash.Hex()]
	_, isArchived := n.archivedTXs[txHash.Hex()]

	if !isAlreadyPending && !isArchived {
		fmt.Printf("Added Pending TX %s from Peer %s\n", txJson, fromPeer.TcpAddress())
		n.pendingTXs[txHash.Hex()] = tx
		n.newPendingTXs <- tx
	}

	return nil
}

func (n *Node) getPendingTXsAsArray() []internal.SignedTx {
	txs := make([]internal.SignedTx, len(n.pendingTXs))

	i := 0
	for _, tx := range n.pendingTXs {
		txs[i] = tx
		i++
	}

	return txs
}

func (n *Node) removeMinedPendingTXs(block internal.Block) {
	if len(block.TXs) > 0 && len(n.pendingTXs) > 0 {
		fmt.Println("Updating in-memory Pending TXs Pool:")
	}

	for _, tx := range block.TXs {
		txHash, _ := tx.Hash()
		if _, exists := n.pendingTXs[txHash.Hex()]; exists {
			fmt.Printf("\t-archiving mined TX: %s\n", txHash.Hex())

			n.archivedTXs[txHash.Hex()] = tx
			delete(n.pendingTXs, txHash.Hex())
		}
	}
}

func (n *Node) LatestBlockHash() internal.Hash {
	return n.state.LatestBlockHash()
}

func (n *Node) AddPeer(peer PeerNode) {
	n.knownPeers[peer.TcpAddress()] = peer
}

func (n *Node) RemovePeer(peer PeerNode) {
	delete(n.knownPeers, peer.TcpAddress())
}

func (n *Node) IsKnownPeer(peer PeerNode) bool {
	if peer.IP == n.info.IP && peer.Port == n.info.Port {
		return true
	}

	_, isKnownPeer := n.knownPeers[peer.TcpAddress()]

	return isKnownPeer
}
