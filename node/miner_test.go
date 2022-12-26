package node

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/rawdaGastan/learn_block_chain/internal"
)

func TestValidBlockHash(t *testing.T) {
	hexHash := "000000fa04f8160395c387277f8b2f14837603383d33809a4db586086168edfa"
	var hash = internal.Hash{}

	hex.Decode(hash[:], []byte(hexHash))

	isValid := internal.IsBlockHashValid(hash)
	if !isValid {
		t.Fatalf("hash '%s' starting with 6 zeroes is suppose to be valid", hexHash)
	}
}

func TestInvalidBlockHash(t *testing.T) {
	hexHash := "000001fa04f8160395c387277f8b2f14837603383d33809a4db586086168edfa"
	var hash = internal.Hash{}

	hex.Decode(hash[:], []byte(hexHash))

	isValid := internal.IsBlockHashValid(hash)
	if isValid {
		t.Fatal("hash is not suppose to be valid")
	}
}

func TestMine(t *testing.T) {
	miner := internal.NewAccount("rawda")
	pendingBlock := createRandomPendingBlock(miner)

	ctx := context.Background()

	minedBlock, err := Mine(ctx, pendingBlock)
	if err != nil {
		t.Fatal(err)
	}

	minedBlockHash, err := minedBlock.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if !internal.IsBlockHashValid(minedBlockHash) {
		t.Fatal()
	}

	if minedBlock.Header.Miner != miner {
		t.Fatal("mined block miner should equal miner from pending block")
	}
}

func TestMineWithTimeout(t *testing.T) {
	miner := internal.NewAccount("rawda")
	pendingBlock := createRandomPendingBlock(miner)

	ctx, _ := context.WithTimeout(context.Background(), time.Microsecond*100)

	_, err := Mine(ctx, pendingBlock)
	if err == nil {
		t.Fatal(err)
	}
}

func createRandomPendingBlock(miner internal.Account) PendingBlock {
	return NewPendingBlock(
		internal.Hash{},
		0,
		miner,
		[]internal.Tx{
			internal.Tx{From: "rawda", To: "babayaga", Value: 1, Time: 1579451695, Data: ""},
		},
	)
}
