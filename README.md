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
  -dotfiles=false: watch on dotfiles
  -ext="": extentions of files to watch: -ext='less,js,coffee'
  -match=".*": file(fullpath) match regexp
  -pidfile="": file that content pid of running process($WWATCH_PID)
  -recursive=false: walk recursive over directories
  -version=false: print version
</pre>

### Example

`wwatch -cmd='go install' -ext='go'`

## Config files

wwatch supports configuration files in [toml](https://github.com/mojombo/toml) format.

### Example of single task

```toml
dir = "./app/assets/styles"
cwd = "."
cmd = "lessc"
args = ["./app/assets/styles/style.less", "./public/style.css"]
ext = "less"
delay = "1s"
```

### Example of multiple tasks

```toml
dir = "./app/assets"
cwd = "."
delay = "1s"

[run.server]
dir = "."
ext = "go"
cmd = "bash"
args = ["-c", "go run *.go"]
pidfile = "tmp/server.pid"

[run.less]
cmd = "lessc"
args = ["./app/assets/styles/style.less", "./public/style.css"]
match = ".*\\.less$"

[run.uglifyjs]
cmd = "uglifyjs"
args = ["app/assets/javascripts/app.js", "-o", "public/app.min.js"]
match = ".*\\.js$"
```
