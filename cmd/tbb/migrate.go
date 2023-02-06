package main

import (
	"context"
	"fmt"
	"time"

	"github.com/rawdaGastan/learn_block_chain/internal"
	"github.com/rawdaGastan/learn_block_chain/node"
	"github.com/spf13/cobra"
)

var migrateCmd = func() *cobra.Command {
	var migrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Migrates the blockchain internal according to new business rules.",
		Run: func(cmd *cobra.Command, args []string) {
			miner, _ := cmd.Flags().GetString(flagMiner)
			ip, _ := cmd.Flags().GetString(flagIP)
			port, _ := cmd.Flags().GetUint64(flagPort)

			peer := node.NewPeerNode(
				ip,
				defaultPort,
				true,
				false,
				internal.NewAccount("rawda"),
			)

			n := node.New(getDataDirFromCmd(cmd), ip, port, internal.NewAccount(miner), peer)
			/*
				n.AddPendingTX(internal.NewTx("rawda", "rawda", 3, ""), peer)
				n.AddPendingTX(internal.NewTx("rawda", "babayaga", 2000, ""), peer)
				n.AddPendingTX(internal.NewTx("babayaga", "rawda", 1, ""), peer)
				n.AddPendingTX(internal.NewTx("babayaga", "caesar", 1000, ""), peer)
				n.AddPendingTX(internal.NewTx("babayaga", "rawda", 50, ""), peer)
			*/
			ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*15)

			go func() {
				ticker := time.NewTicker(time.Second * 10)

				for {
					select {
					case <-ticker.C:
						if !n.LatestBlockHash().IsEmpty() {
							closeNode()
							return
						}
					}
				}
			}()

			err := n.Run(ctx)
			if err != nil {
				fmt.Println(err)
			}

		},
	}

	addDefaultRequiredFlags(migrateCmd)
	migrateCmd.Flags().String(flagMiner, "", "miner account")
	migrateCmd.MarkFlagRequired(flagMiner)

	migrateCmd.Flags().Uint64(flagPort, 8080, "port")
	migrateCmd.MarkFlagRequired(flagPort)

	migrateCmd.Flags().String(flagIP, "127.0.0.1", "ip")

	return migrateCmd
}
