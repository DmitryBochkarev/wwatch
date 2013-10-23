## Installation

`go get -u github.com/DmitryBochkarev/wwatch`

## Usage

<pre>
$ wwatch -h

Usage of wwatch:
  -after=false: run command only after files changed
  -cmd="": command to run, rerun on file changed
  -config="": path to configuration file(*.toml)
  -cwd=".": current working directory
  -delay="100ms": delay before rerun cmd
  -dir=".": directory to watch
  -dotfiles=false: watch on dotfiles
  -ext="": extentions of files to watch: -ext='less,js,coffee'
  -ignore="": regexp patter for ignore watch
  -match=".*": file(fullpath) match regexp
  -onstart="": command to run on start
  -pidfile="": file that content pid of running process
  -recursive=false: walk recursive over directories
  -version=false: print version
</pre>

### Example

`wwatch -cmd='go install' -ext='go'`

## Config files

wwatch supports configuration files in [toml](https://github.com/mojombo/toml) format.

### Example of single task

```toml
cwd = "." #relative to config file
cmd = "lessc"
args = ["./app/assets/styles/style.less", "./public/style.css"]
ext = ["less"]
delay = "1s"
```

### Example of multiple tasks

```toml
cwd = "."
delay = "1s"
ignore = "^~.*$" #vim files
before = "bash"
before_args = ["-c", "rm -rf ./tmp/*"]

[run.server]
ext = "go"
cmd = "bash"
args = ["-c", "go run *.go"]
pidfile = "tmp/server.pid"

[run.less]
dir = "./app/assets/styles"
cmd = "lessc"
args = ["./app/assets/styles/style.less", "./public/style.css"]
match = ".*\\.less$" #same as ext=["less"]

[run.uglifyjs]
dir = "./app/assets/javascripts"
cmd = "uglifyjs"
args = ["app/assets/javascripts/app.js", "-o", "public/app.min.js"]
ext = ["js"]
```
