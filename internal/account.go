package internal

type Account string

func NewAccount(value string) Account {
	return Account(value)
}
