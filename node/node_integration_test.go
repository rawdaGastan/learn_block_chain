package node

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rawdaGastan/learn_block_chain/internal"
	"github.com/rawdaGastan/learn_block_chain/wallet"
)

// The password for testing keystore files:
//
//	./test_rawda--3eb92807f1f91a8d4d85bc908c7f86dcddb1df57
//	./test_babayaga--6fdc0d8d15ae6b4ebf45c52fd2aafbcbb19a65c8
//
// Pre-generated for testing purposes using wallet_test.go.
//
// It's necessary to have pre-existing accounts before a new node
// with fresh new, empty keystore is initialized and booted in order
// to configure the accounts balances in genesis.json
//
// I.e: A quick solution to a chicken-egg problem.
const testKsRawdaAccount = "0x3eb92807f1f91a8d4d85bc908c7f86dcddb1df57"
const testKsBabaYagaAccount = "0x6fdc0d8d15ae6b4ebf45c52fd2aafbcbb19a65c8"
const testKsRawdaFile = "test_rawda--3eb92807f1f91a8d4d85bc908c7f86dcddb1df57"
const testKsBabaYagaFile = "test_babayaga--6fdc0d8d15ae6b4ebf45c52fd2aafbcbb19a65c8"
const testKsAccountsPwd = "security123"

func TestNode_Run(t *testing.T) {
	datadir, err := getTestDataDirPath()
	if err != nil {
		t.Fatal(err)
	}
	err = internal.RemoveDir(datadir)
	if err != nil {
		t.Fatal(err)
	}

	n := New(datadir, "127.0.0.1", 8085, internal.NewAccount(DefaultMiner), PeerNode{})

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	err = n.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNode_Mining(t *testing.T) {
	dataDir, rawda, babaYaga, err := setupTestNodeDir()
	if err != nil {
		t.Error(err)
	}
	defer internal.RemoveDir(dataDir)

	// Required for AddPendingTX() to describe
	// from what node the TX came from (local node in this case)
	nInfo := NewPeerNode(
		"127.0.0.1",
		8085,
		false,
		true,
		babaYaga,
	)

	// Construct a new Node instance and configure
	// Rawda as a miner
	n := New(dataDir, nInfo.IP, nInfo.Port, rawda, nInfo)

	// Allow the mining to run for 30 mins, in the worst case
	ctx, closeNode := context.WithTimeout(
		context.Background(),
		time.Minute*30,
	)

	// Schedule a new TX in 3 seconds from now, in a separate thread
	// because the n.Run() few lines below is a blocking call
	go func() {
		time.Sleep(time.Second * miningIntervalSeconds / 3)

		tx := internal.NewTx(rawda, babaYaga, 1, 1, "")
		signedTx, err := wallet.SignTxWithKeystoreAccount(tx, rawda, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
		if err != nil {
			t.Error(err)
			return
		}

		_ = n.AddPendingTX(signedTx, nInfo)
	}()

	// Schedule a new TX in 12 seconds from now simulating
	// that it came in - while the first TX is being mined
	go func() {
		time.Sleep(time.Second*miningIntervalSeconds + 2)

		tx := internal.NewTx(rawda, babaYaga, 2, 2, "")
		signedTx, err := wallet.SignTxWithKeystoreAccount(tx, rawda, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
		if err != nil {
			t.Error(err)
			return
		}

		_ = n.AddPendingTX(signedTx, nInfo)
	}()

	go func() {
		// Periodically check if we mined the 2 blocks
		ticker := time.NewTicker(10 * time.Second)

		for {
			select {
			case <-ticker.C:
				if n.state.LatestBlock().Header.Number == 1 {
					closeNode()
					return
				}
			}
		}
	}()

	// Run the node, mining and everything in a blocking call (hence the go-routines before)
	_ = n.Run(ctx)

	if n.state.LatestBlock().Header.Number != 1 {
		t.Fatal("2 pending TX not mined into 2 under 30m")
	}
}

// Expect:
//
//	ERROR: wrong TX. Sender '0x3EB9....' is forged
//
// TODO: Improve this with TX Receipt concept in next chapters.
// TODO: Improve this with a 100% clear error check.
func TestNode_ForgedTx(t *testing.T) {
	dataDir, rawda, babaYaga, err := setupTestNodeDir()
	if err != nil {
		t.Error(err)
	}
	defer internal.RemoveDir(dataDir)

	n := New(dataDir, "127.0.0.1", 8085, rawda, PeerNode{})
	ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*30)
	rawdaPeerNode := NewPeerNode("127.0.0.1", 8085, false, true, rawda)

	txValue := uint(5)
	txNonce := uint(1)
	tx := internal.NewTx(rawda, babaYaga, txValue, txNonce, "")

	validSignedTx, err := wallet.SignTxWithKeystoreAccount(tx, rawda, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
	if err != nil {
		t.Error(err)
		return
	}

	err = n.AddPendingTX(validSignedTx, rawdaPeerNode)
	if err != nil {
		t.Error(err)
		return
	}

	go func() {
		ticker := time.NewTicker(time.Second * (miningIntervalSeconds - 3))
		wasForgedTxAdded := false

		for {
			select {
			case <-ticker.C:
				if !n.state.LatestBlockHash().IsEmpty() {
					if wasForgedTxAdded && !n.isMining {
						closeNode()
						return
					}

					if !wasForgedTxAdded {
						// Attempt to forge the same TX but with modified time
						// Because the TX.time changed, the TX.signature will be considered forged
						// internal.NewTx() changes the TX time
						forgedTx := internal.NewTx(rawda, babaYaga, txValue, txNonce, "")
						// Use the signature from a valid TX
						forgedSignedTx := internal.NewSignedTx(forgedTx, validSignedTx.Sig)

						_ = n.AddPendingTX(forgedSignedTx, rawdaPeerNode)
						wasForgedTxAdded = true

						time.Sleep(time.Second * (miningIntervalSeconds + 3))
					}
				}
			}
		}
	}()

	_ = n.Run(ctx)

	if n.state.LatestBlock().Header.Number != 0 {
		t.Fatal("was suppose to mine only one TX. The second TX was forged")
	}

	if n.state.Balances[babaYaga] != txValue {
		t.Fatal("forged tx succeeded")
	}
}

// Expect:
//
//	ERROR: wrong TX. Sender '0x3EB9...' next nonce must be '2', not '1'
//
// TODO: Improve this with TX Receipt concept in next chapters.
// TODO: Improve this with a 100% clear error check.
func TestNode_ReplayedTx(t *testing.T) {
	dataDir, rawda, babaYaga, err := setupTestNodeDir()
	if err != nil {
		t.Error(err)
	}
	defer internal.RemoveDir(dataDir)

	n := New(dataDir, "127.0.0.1", 8085, rawda, PeerNode{})
	ctx, closeNode := context.WithCancel(context.Background())
	rawdaPeerNode := NewPeerNode("127.0.0.1", 8085, false, true, rawda)
	babaYagaPeerNode := NewPeerNode("127.0.0.1", 8086, false, true, babaYaga)

	txValue := uint(5)
	txNonce := uint(1)
	tx := internal.NewTx(rawda, babaYaga, txValue, txNonce, "")

	signedTx, err := wallet.SignTxWithKeystoreAccount(tx, rawda, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
	if err != nil {
		t.Error(err)
		return
	}

	_ = n.AddPendingTX(signedTx, rawdaPeerNode)

	go func() {
		ticker := time.NewTicker(time.Second * (miningIntervalSeconds - 3))
		wasReplayedTxAdded := false

		for {
			select {
			case <-ticker.C:
				if !n.state.LatestBlockHash().IsEmpty() {
					if wasReplayedTxAdded && !n.isMining {
						closeNode()
						return
					}

					// The Rawda's original TX got mined.
					// Execute the attack by replaying the TX again!
					if !wasReplayedTxAdded {
						// Simulate the TX was submitted to different node
						n.archivedTXs = make(map[string]internal.SignedTx)
						// Execute the attack
						_ = n.AddPendingTX(signedTx, babaYagaPeerNode)
						wasReplayedTxAdded = true

						time.Sleep(time.Second * (miningIntervalSeconds + 3))
					}
				}
			}
		}
	}()

	_ = n.Run(ctx)

	if n.state.Balances[babaYaga] == txValue*2 {
		t.Errorf("replayed attack was successful :( Damn digital signatures!")
		return
	}

	if n.state.Balances[babaYaga] != txValue {
		t.Errorf("replayed attack was successful :( Damn digital signatures!")
		return
	}

	if n.state.LatestBlock().Header.Number == 1 {
		t.Errorf("the second block was not suppose to be persisted because it contained a malicious TX")
		return
	}
}

// The test logic summary:
//   - BabaYaga runs the node
//   - BabaYaga tries to mine 2 TXs
//   - The mining gets interrupted because a new block from Rawda gets synced
//   - Rawda will get the block reward for this synced block
//   - The synced block contains 1 of the TXs BabaYaga tried to mine
//   - BabaYaga tries to mine 1 TX left
//   - BabaYaga succeeds and gets her block reward
func TestNode_MiningStopsOnNewSyncedBlock(t *testing.T) {
	babaYaga := internal.NewAccount(testKsBabaYagaAccount)
	rawda := internal.NewAccount(testKsRawdaAccount)

	dataDir, err := getTestDataDirPath()
	if err != nil {
		t.Fatal(err)
	}

	genesisBalances := make(map[common.Address]uint)
	genesisBalances[rawda] = 1000000
	genesis := internal.Genesis{Balances: genesisBalances}
	genesisJson, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}

	err = internal.InitDataDirIfNotExists(dataDir, genesisJson)
	defer internal.RemoveDir(dataDir)

	err = copyKeystoreFilesIntoTestDataDirPath(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	// Required for AddPendingTX() to describe
	// from what node the TX came from (local node in this case)
	nInfo := NewPeerNode(
		"127.0.0.1",
		8085,
		false,
		true,
		internal.NewAccount(""),
	)

	n := New(dataDir, nInfo.IP, nInfo.Port, babaYaga, nInfo)

	// Allow the test to run for 30 mins, in the worst case
	ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*30)

	tx1 := internal.NewTx(rawda, babaYaga, 1, 1, "")
	tx2 := internal.NewTx(rawda, babaYaga, 2, 2, "")

	signedTx1, err := wallet.SignTxWithKeystoreAccount(tx1, rawda, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
	if err != nil {
		t.Error(err)
		return
	}

	signedTx2, err := wallet.SignTxWithKeystoreAccount(tx2, rawda, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
	if err != nil {
		t.Error(err)
		return
	}
	tx2Hash, err := signedTx2.Hash()
	if err != nil {
		t.Error(err)
		return
	}

	// Pre-mine a valid block without running the `n.Run()`
	// with Rawda as a miner who will receive the block reward,
	// to simulate the block came on the fly from another peer
	validPreMinedPb := NewPendingBlock(internal.Hash{}, 0, rawda, []internal.SignedTx{signedTx1})
	validSyncedBlock, err := Mine(ctx, validPreMinedPb)
	if err != nil {
		t.Fatal(err)
	}

	// Add 2 new TXs into the BabaYaga's node, triggers mining
	go func() {
		time.Sleep(time.Second * (miningIntervalSeconds - 2))

		err := n.AddPendingTX(signedTx1, nInfo)
		if err != nil {
			t.Fatal(err)
		}

		err = n.AddPendingTX(signedTx2, nInfo)
		if err != nil {
			t.Fatal(err)
		}
	}()

	// TODO: Fix a race condition when the block gets mined
	//       before the validBlock gets synced.
	//
	// Interrupt the previously started mining with a new synced block
	// BUT this block contains only 1 TX the previous mining activity tried to mine
	// which means the mining will start again for the one pending TX that is left and wasn't in
	// the synced block
	go func() {
		time.Sleep(time.Second * (miningIntervalSeconds + 2))
		if !n.isMining {
			t.Fatal("should be mining")
		}

		_, err := n.state.AddBlock(validSyncedBlock)
		if err != nil {
			t.Fatal(err)
		}
		// Mock the Rawda's block came from a network
		n.newSyncedBlocks <- validSyncedBlock

		time.Sleep(time.Second * 2)
		if n.isMining {
			t.Fatal("synced block should have canceled mining")
		}

		// Mined TX1 by Rawda should be removed from the Mempool
		_, onlyTX2IsPending := n.pendingTXs[tx2Hash.Hex()]

		if len(n.pendingTXs) != 1 && !onlyTX2IsPending {
			t.Fatal("synced block should have canceled mining of already mined TX")
		}

		time.Sleep(time.Second * (miningIntervalSeconds + 2))
		if !n.isMining {
			t.Fatal("should be mining again the 1 TX not included in synced block")
		}
	}()

	go func() {
		// Regularly check whenever both TXs are now mined
		ticker := time.NewTicker(time.Second * 10)

		for {
			select {
			case <-ticker.C:
				if n.state.LatestBlock().Header.Number == 1 {
					closeNode()
					return
				}
			}
		}
	}()

	go func() {
		time.Sleep(time.Second * 2)

		// Take a snapshot of the DB balances
		// before the mining is finished and the 2 blocks
		// are created.
		startingRawdaBalance := n.state.Balances[rawda]
		startingBabaYagaBalance := n.state.Balances[babaYaga]

		// Wait until the 30 mins timeout is reached or
		// the 2 blocks got already mined and the closeNode() was triggered
		<-ctx.Done()

		endRawdaBalance := n.state.Balances[rawda]
		endBabaYagaBalance := n.state.Balances[babaYaga]

		// In TX1 Rawda transferred 1 TBB token to BabaYaga
		// In TX2 Rawda transferred 2 TBB tokens to BabaYaga
		expectedEndRawdaBalance := startingRawdaBalance - tx1.Value - tx2.Value + internal.BlockReward
		expectedEndBabaYagaBalance := startingBabaYagaBalance + tx1.Value + tx2.Value + internal.BlockReward

		if endRawdaBalance != expectedEndRawdaBalance {
			t.Fatalf("Rawda expected end balance is %d not %d", expectedEndRawdaBalance, endRawdaBalance)
		}

		if endBabaYagaBalance != expectedEndBabaYagaBalance {
			t.Fatalf("BabaYaga expected end balance is %d not %d", expectedEndBabaYagaBalance, endBabaYagaBalance)
		}

		t.Logf("Starting Rawda balance: %d", startingRawdaBalance)
		t.Logf("Starting BabaYaga balance: %d", startingBabaYagaBalance)
		t.Logf("Ending Rawda balance: %d", endRawdaBalance)
		t.Logf("Ending BabaYaga balance: %d", endBabaYagaBalance)
	}()

	_ = n.Run(ctx)

	if n.state.LatestBlock().Header.Number != 1 {
		t.Fatal("was suppose to mine 2 pending TX into 2 valid blocks under 30m")
	}

	if len(n.pendingTXs) != 0 {
		t.Fatal("no pending TXs should be left to mine")
	}
}

// Creates dir like: "/tmp/tbb_test945924586"
func getTestDataDirPath() (string, error) {
	return ioutil.TempDir(os.TempDir(), "tbb_test")
}

// Copy the pre-generated, commited keystore files from this folder into the new testDataDirPath()
//
// Afterwards the test datadir path will look like:
//
//	"/tmp/tbb_test945924586/keystore/test_rawda--3eb92807f1f91a8d4d85bc908c7f86dcddb1df57"
//	"/tmp/tbb_test945924586/keystore/test_babayaga--6fdc0d8d15ae6b4ebf45c52fd2aafbcbb19a65c8"
func copyKeystoreFilesIntoTestDataDirPath(dataDir string) error {
	rawdaSrcKs, err := os.Open(testKsRawdaFile)
	if err != nil {
		return err
	}
	defer rawdaSrcKs.Close()

	ksDir := filepath.Join(wallet.GetKeystoreDirPath(dataDir))

	err = os.Mkdir(ksDir, 0777)
	if err != nil {
		return err
	}

	rawdaDstKs, err := os.Create(filepath.Join(ksDir, testKsRawdaFile))
	if err != nil {
		return err
	}
	defer rawdaDstKs.Close()

	_, err = io.Copy(rawdaDstKs, rawdaSrcKs)
	if err != nil {
		return err
	}

	babayagaSrcKs, err := os.Open(testKsBabaYagaFile)
	if err != nil {
		return err
	}
	defer babayagaSrcKs.Close()

	babayagaDstKs, err := os.Create(filepath.Join(ksDir, testKsBabaYagaFile))
	if err != nil {
		return err
	}
	defer babayagaDstKs.Close()

	_, err = io.Copy(babayagaDstKs, babayagaSrcKs)
	if err != nil {
		return err
	}

	return nil
}

// setupTestNodeDir creates a default testing node directory with 2 keystore accounts
//
// Remember to remove the dir once test finishes: defer fs.RemoveDir(dataDir)
func setupTestNodeDir() (dataDir string, rawda, babaYaga common.Address, err error) {
	babaYaga = internal.NewAccount(testKsBabaYagaAccount)
	rawda = internal.NewAccount(testKsRawdaAccount)

	dataDir, err = getTestDataDirPath()
	if err != nil {
		return "", common.Address{}, common.Address{}, err
	}

	genesisBalances := make(map[common.Address]uint)
	genesisBalances[rawda] = 1000000
	genesis := internal.Genesis{Balances: genesisBalances}
	genesisJson, err := json.Marshal(genesis)
	if err != nil {
		return "", common.Address{}, common.Address{}, err
	}

	err = internal.InitDataDirIfNotExists(dataDir, genesisJson)
	if err != nil {
		return "", common.Address{}, common.Address{}, err
	}

	err = copyKeystoreFilesIntoTestDataDirPath(dataDir)
	if err != nil {
		return "", common.Address{}, common.Address{}, err
	}

	return dataDir, rawda, babaYaga, nil
}
