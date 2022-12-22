package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"
)

type State struct {
	Balances        map[Account]uint
	txMempool       []Tx
	dbFile          *os.File
	latestBlockHash Hash
	latestBlock     Block
	hasGenesisBlock bool
}

func NewStateFromDisk(dataDir string) (*State, error) {
	err := InitDataDirIfNotExists(dataDir)
	if err != nil {
		return nil, err
	}

	gen, err := loadGenesis(getGenesisJsonFilePath(dataDir))
	if err != nil {
		return nil, err
	}
	balances := make(map[Account]uint)
	for account, balance := range gen.Balances {
		balances[account] = balance

	}

	dbFilepath := getBlocksDbFilePath(dataDir)
	f, err := os.OpenFile(dbFilepath, os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(f)
	state := &State{balances, make([]Tx, 0), f, Hash{}, Block{}, false}

	// Iterate over each the block.db file's line
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		blockFsJson := scanner.Bytes()
		var blockFs BlockFS
		err = json.Unmarshal(blockFsJson, &blockFs)
		if err != nil {
			return nil, err
		}

		err = state.applyBlock(blockFs.Value)
		if err != nil {
			return nil, err
		}

		state.latestBlockHash = blockFs.Key
		state.latestBlock = blockFs.Value
	}

	return state, nil
}

func (s *State) apply(tx Tx) error {
	if tx.IsReward() {
		s.Balances[tx.To] += tx.Value
		return nil
	}
	if tx.Value > s.Balances[tx.From] {
		return fmt.Errorf("insufficient balance")
	}
	s.Balances[tx.From] -= tx.Value
	s.Balances[tx.To] += tx.Value
	return nil
}

func (s *State) applyBlock(b Block) error {
	for _, tx := range b.TXs {
		if err := s.apply(tx); err != nil {
			return err
		}
	}

	return nil
}

func (s *State) AddBlocks(blocks []Block) error {
	for _, b := range blocks {
		_, err := s.AddBlock(b)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *State) AddBlock(b Block) (Hash, error) {
	pendingState := s.copy()

	err := applyBlock(b, pendingState)
	if err != nil {
		return Hash{}, err
	}

	blockHash, err := b.Hash()
	if err != nil {
		return Hash{}, err
	}

	blockFs := BlockFS{blockHash, b}

	blockFsJson, err := json.Marshal(blockFs)
	if err != nil {
		return Hash{}, err
	}

	fmt.Printf("Persisting new Block to disk:\n")
	fmt.Printf("\t%s\n", blockFsJson)

	_, err = s.dbFile.Write(append(blockFsJson, '\n'))
	if err != nil {
		return Hash{}, err
	}

	s.Balances = pendingState.Balances
	s.latestBlockHash = blockHash
	s.latestBlock = b
	s.hasGenesisBlock = true

	return blockHash, nil
}

// applyBlock verifies if block can be added to the blockchain.
// Block metadata are verified as well as transactions within (sufficient balances, etc).
func applyBlock(b Block, s State) error {
	nextExpectedBlockNumber := s.latestBlock.Header.Number + 1

	if s.hasGenesisBlock && b.Header.Number != nextExpectedBlockNumber {
		return fmt.Errorf("next expected block must be '%d' not '%d'", nextExpectedBlockNumber, b.Header.Number)
	}

	if s.hasGenesisBlock && s.latestBlock.Header.Number > 0 && !reflect.DeepEqual(b.Header.Parent, s.latestBlockHash) {
		return fmt.Errorf("next block parent hash must be '%x' not '%x'", s.latestBlockHash, b.Header.Parent)
	}

	return applyTXs(b.TXs, &s)
}

func (s *State) NextBlockNumber() uint64 {
	return s.latestBlock.Header.Number
}

func (s *State) AddTx(tx Tx) error {
	if err := s.apply(tx); err != nil {
		return err
	}

	s.txMempool = append(s.txMempool, tx)
	return nil
}

func applyTXs(txs []Tx, s *State) error {
	for _, tx := range txs {
		err := applyTx(tx, s)
		if err != nil {
			return err
		}
	}

	return nil
}

func applyTx(tx Tx, s *State) error {
	if tx.IsReward() {
		s.Balances[tx.To] += tx.Value
		return nil
	}

	if tx.Value > s.Balances[tx.From] {
		return fmt.Errorf("wrong TX. Sender '%s' balance is %d TBB. Tx cost is %d TBB", tx.From, s.Balances[tx.From], tx.Value)
	}

	s.Balances[tx.From] -= tx.Value
	s.Balances[tx.To] += tx.Value

	return nil
}

func (s *State) Persist() (Hash, error) {
	// Create a new Block with ONLY the new TXs
	block := NewBlock(
		s.latestBlockHash,
		uint64(time.Now().Unix()),
		s.latestBlock.Header.Number+1,
		s.txMempool,
	)

	blockHash, err := block.Hash()
	if err != nil {
		return Hash{}, err
	}

	blockFs := BlockFS{blockHash, block}

	blockFsJson, err := json.Marshal(blockFs)
	if err != nil {
		return Hash{}, err
	}

	fmt.Printf("Persisting new Block to disk:\n")
	fmt.Printf("\t%s\n", blockFsJson)

	// Write it to the DB file on a new line
	_, err = s.dbFile.Write(append(blockFsJson, '\n'))
	if err != nil {
		return Hash{}, err
	}
	s.latestBlockHash = blockHash

	// Reset the mempool
	s.txMempool = []Tx{}
	return s.latestBlockHash, nil
}

func (s *State) Close() {
	s.dbFile.Close()
}

func (s *State) LatestBlockHash() Hash {
	return s.latestBlockHash
}

func (s *State) LatestBlock() Block {
	return s.latestBlock
}

func (s *State) copy() State {
	c := State{}
	c.hasGenesisBlock = s.hasGenesisBlock
	c.latestBlock = s.latestBlock
	c.latestBlockHash = s.latestBlockHash
	c.txMempool = make([]Tx, len(s.txMempool))
	c.Balances = make(map[Account]uint)

	for acc, balance := range s.Balances {
		c.Balances[acc] = balance
	}

	c.txMempool = append(c.txMempool, s.txMempool...)

	return c
}
