[English](README.md) | [繁體中文](README.zh-TW.md)

# Task Panel

Task Panel (TP) is a simple terminal UI for quickly starting and stopping
processes.

Think of it as a simple take on systemd.

I have plenty of programs I need to start and stop throughout the day, and the
commands often come with a bunch of arguments that are hard to remember. I
used to make aliases for them, but as the list grew, I realized they all worked
in roughly the same way.

So, I put the start and stop commands in a TOML file and let Task Panel run
them from one place.

In other words, Task Panel does not try to be a full process manager. It is a
handy interface for managing a collection of start and stop commands.

## Run

1. Use [config.example.toml](config.example.toml) as a guide, then create your
   configuration file at `~/.config/task_panel/config.toml`.
   - `config.example.toml` also defines the configuration format.
1. Download the file for your platform from
   [Releases](../../releases), unpack it, and run the binary.

## Demo

Here is a sample configuration:

```toml
show_start_confirmation=false
show_stop_confirmation=false
[[process]]
name = "My Wiki by Zensical"
start = "zensical serve -f $HOME/Documents/wikidata/zensical.toml -a localhost:42001"
id = "pgrep -f 'zensical serve'"

[[process]]
name = "Static Web Server"
start = 'container run -d --rm --name web_server -p 42000:80 -v "$HOME/Documents/wikidata/site:/public" joseluisq/static-web-server:latest'
stop = "container stop web_server"
find = "container inspect web_server |grep  running"
```

Then:

1. Start Task Panel.
1. Check that Zensical is not running.
1. Start Zensical and use `ps` to confirm it is running.
1. Use the container command to check that the web server is not running.
1. Start the web server with the container command, then use `container ls` to
   confirm it is running.
1. Stop the web server, then use `container ls` to confirm it has stopped.

![Demo](docs/imgs/demo.gif)

## Development

- Go: `1.26.0`

```bash
$ make fmt
$ make test
$ make build

$ make # build the binary
```

## Release

- The version is defined by `applicationVersion` in `task_panel.go`.
- This project uses GitHub Actions. Create a version tag beginning with `v` to
  publish a release.

```sh
git tag -a v0.1.0 -m "Release 0.1.0"
git push origin v0.1.0
```
