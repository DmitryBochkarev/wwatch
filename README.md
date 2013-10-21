## Installation

`go get -u github.com/DmitryBochkarev/wwatch`

## Usage

`wwatch -dir='.' -cmd='go run *.go' -match='.*\.go$'`

## TODO

- config file
- after kill callback
- custom kill command `kill -9 $WATCH_PID`
