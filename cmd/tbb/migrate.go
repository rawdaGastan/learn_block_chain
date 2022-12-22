package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rawdaGastan/learn_block_chain/internal"
	"github.com/spf13/cobra"
)

var migrateCmd = func() *cobra.Command {
	var migrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Migrates the blockchain internal according to new business rules.",
		Run: func(cmd *cobra.Command, args []string) {
			state, err := internal.NewStateFromDisk(getDataDirFromCmd(cmd))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			defer state.Close()

			block0 := internal.NewBlock(
				internal.Hash{},
				state.NextBlockNumber(),
				uint64(time.Now().Unix()),
				[]internal.Tx{
					internal.NewTx("andrej", "andrej", 3, ""),
					internal.NewTx("andrej", "andrej", 700, "reward"),
				},
			)

			block0hash, err := state.AddBlock(block0)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			block1 := internal.NewBlock(
				block0hash,
				state.NextBlockNumber(),
				uint64(time.Now().Unix()),
				[]internal.Tx{
					internal.NewTx("andrej", "babayaga", 2000, ""),
					internal.NewTx("andrej", "andrej", 100, "reward"),
					internal.NewTx("babayaga", "andrej", 1, ""),
					internal.NewTx("babayaga", "caesar", 1000, ""),
					internal.NewTx("babayaga", "andrej", 50, ""),
					internal.NewTx("andrej", "andrej", 600, "reward"),
				},
			)

			block1hash, err := state.AddBlock(block1)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			block2 := internal.NewBlock(
				block1hash,
				state.NextBlockNumber(),
				uint64(time.Now().Unix()),
				[]internal.Tx{
					internal.NewTx("andrej", "andrej", 24700, "reward"),
				},
			)

			_, err = state.AddBlock(block2)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}

	addDefaultRequiredFlags(migrateCmd)

	return migrateCmd
}
