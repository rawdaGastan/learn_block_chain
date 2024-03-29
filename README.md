# Learn block chain

This is the code example for Lukas Lukac build a Blockchain from Scratch in Go book.

:books: Download the eBook from: [https://web3.coach#book](https://web3.coach#book)

:mag: Find the repository [here](https://github.com/web3coach/the-blockchain-bar)

:pushpin: Find the tutorial [here](https://www.freecodecamp.org/news/build-a-blockchain-in-golang-from-scratch/)

## Install

- `go install ./cmd/tbb/...`
- `export PATH=/home/rawda/go/bin/:$PATH`

## Use

- `tbb balances list`
- `tbb migrate --datadir=data`
- `tbb run --port=8080 --datadir=data`

## Testing

- `go test -timeout=0 -count=1 ./node -test.v -test.run ^TestNode_Mining$`
- `go test -timeout=0 -count=1 ./node -test.v -test.run ^TestNode_MiningStopsOnNewSyncedBlock$`
- `go test ./node -timeout=0 -test.v -test.run ^TestNode_ForgedTx$`
