## Installation

`go get -u github.com/DmitryBochkarev/wwatch`

## Usage

<pre>
  -cmd="": command to run
  -cwd=".": current working directory
  -delay=100ms: delay before rerun cmd
  -dir=".": directory to watch
  -kill="": command to shutdown process. Example: kill -9 $WWATCH_PID
  -match=".*": file(fullpath) match regexp
  -version=false: print version
</pre>

## Example

`wwatch -cmd='go run *.go' -match='.*\.go$'`

## TODO

- config file
