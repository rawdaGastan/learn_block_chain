package node

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rawdaGastan/learn_block_chain/internal"
	"github.com/rawdaGastan/learn_block_chain/wallet"
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
	minerPrivKey, _, miner, err := generateKey()
	if err != nil {
		t.Fatal(err)
	}

	pendingBlock, err := createRandomPendingBlock(minerPrivKey, miner)
	if err != nil {
		t.Fatal(err)
	}

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

	if minedBlock.Header.Miner.String() != miner.String() {
		t.Fatal("mined block miner should equal miner from pending block")
	}
}

func TestMineWithTimeout(t *testing.T) {
	minerPrivKey, _, miner, err := generateKey()
	if err != nil {
		t.Fatal(err)
	}

	pendingBlock, err := createRandomPendingBlock(minerPrivKey, miner)
	if err != nil {
		t.Fatal(err)
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Microsecond*100)

	_, err = Mine(ctx, pendingBlock)
	if err == nil {
		t.Fatal(err)
	}
}

func generateKey() (*ecdsa.PrivateKey, ecdsa.PublicKey, common.Address, error) {
	privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return nil, ecdsa.PublicKey{}, common.Address{}, err
	}

	pubKey := privKey.PublicKey
	pubKeyBytes := elliptic.Marshal(crypto.S256(), pubKey.X, pubKey.Y)
	pubKeyBytesHash := crypto.Keccak256(pubKeyBytes[1:])

	account := common.BytesToAddress(pubKeyBytesHash[12:])

	return privKey, pubKey, account, nil
}

func createRandomPendingBlock(privKey *ecdsa.PrivateKey, acc common.Address) (PendingBlock, error) {
	tx := internal.NewTx(acc, internal.NewAccount(testKsBabaYagaAccount), 1, 1, "")
	signedTx, err := wallet.SignTx(tx, privKey)
	if err != nil {
		return PendingBlock{}, err
	}

	return NewPendingBlock(
		internal.Hash{},
		0,
		acc,
		[]internal.SignedTx{signedTx},
	), nil
}
