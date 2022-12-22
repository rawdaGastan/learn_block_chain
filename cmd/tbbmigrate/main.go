package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rawdaGastan/learn_block_chain/internal"
)

func main() {
	state, err := internal.NewStateFromDisk()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer state.Close()

	block0 := internal.NewBlock(
		internal.Hash{},
		uint64(time.Now().Unix()),
		[]internal.Tx{
			internal.NewTx("rawda", "rawda", 3, ""),
			internal.NewTx("rawda", "rawda", 700, "reward"),
		},
	)

	state.AddBlock(block0)
	block0hash, _ := state.Persist()

	block1 := internal.NewBlock(
		block0hash,
		uint64(time.Now().Unix()),
		[]internal.Tx{
			internal.NewTx("rawda", "babayaga", 2000, ""),
			internal.NewTx("rawda", "rawda", 100, "reward"),
			internal.NewTx("babayaga", "rawda", 1, ""),
			internal.NewTx("babayaga", "caesar", 1000, ""),
			internal.NewTx("babayaga", "rawda", 50, ""),
			internal.NewTx("rawda", "rawda", 600, "reward"),
		},
	)

	state.AddBlock(block1)
	state.Persist()
}
