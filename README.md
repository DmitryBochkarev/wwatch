## Installation

`go get -u github.com/DmitryBochkarev/wwatch`

## Usage

<pre>
$ wwatch -h

Usage of wwatch:
  -cmd="": command to run
  -config="": path to configuration file(*.toml)
  -cwd=".": current working directory
  -delay="100ms": delay before rerun cmd
  -dir=".": directory to watch
  -kill="": command to shutdown process. Example: kill -9 $WWATCH_PID. Default send INT signal.
  -match=".*": file(fullpath) match regexp
  -version=false: print version
</pre>

### Example

`wwatch -cmd='go run *.go' -match='.*\.go$'`

## Config files

wwatch supports configuration files in [toml](https://github.com/mojombo/toml) format.

Each task may have next fields:

```toml
dir = "<directory to watch(relative to configuration file or absolute)>"
cwd = "<working directory for task(relative to configuration file or absolute)>"
cmd = "<binary name>"
args = ["<array>", "<of>", "<arguments>"]
kill = "<binary called for kill task>"
kill_args = ["<array>", "<of>", "<arguments>", "<passed>", "<to>", "<kill>"]
match = "<string compiled to regexp>"
delay = "<string repsented delay before kill/rerun>"
```

### Example of single task

```toml
dir = "./app/assets/styles"
cwd = "."
cmd = "lessc"
args = ["./app/assets/styles/style.less", "./public/style.css"]
kill = "kill"
kill_args = ["-9", "$WWATCH_PID"]
match = ".*\\.less$"
delay = "1s"
```

### Example of multiple tasks

```toml
dir = "./app/assets"
cwd = "."
kill = "kill"
kill_args = ["-9", "$WWATCH_PID"]
delay = "1s"

[run.less]
cmd = "lessc"
args = ["./app/assets/styles/style.less", "./public/style.css"]
match = ".*\\.less$"

[run.uglifyjs]
cmd = "uglifyjs"
args = ["app/assets/javascripts/app.js", "-o", "public/app.min.js"]
match = ".*\\.js$"
```
