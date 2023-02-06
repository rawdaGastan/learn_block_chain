package internal

import "github.com/ethereum/go-ethereum/common"

type Account string

func NewAccount(value string) common.Address {
	return common.HexToAddress(value)
}
