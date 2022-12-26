package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rawdaGastan/learn_block_chain/internal"
	"github.com/rawdaGastan/learn_block_chain/node"
	"github.com/spf13/cobra"
)

var defaultPort uint64 = 8080

func runCmd() *cobra.Command {
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Launches the TBB node and its HTTP API.",
		Run: func(cmd *cobra.Command, args []string) {
			miner, _ := cmd.Flags().GetString(flagMiner)
			ip, _ := cmd.Flags().GetString(flagIP)
			port, _ := cmd.Flags().GetUint64(flagPort)

			fmt.Println("Launching TBB node and its HTTP API...")

			bootstrap := node.NewPeerNode(
				"127.0.0.1",
				8080,
				true,
				false,
				internal.NewAccount("rawda"),
			)

			n := node.New(getDataDirFromCmd(cmd), ip, port, internal.NewAccount(miner), bootstrap)
			err := n.Run(context.Background())
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}

	addDefaultRequiredFlags(runCmd)

	runCmd.Flags().Uint64(flagPort, 8080, "port")
	runCmd.MarkFlagRequired(flagPort)

	runCmd.Flags().String(flagIP, "127.0.0.1", "ip")

	return runCmd
}
