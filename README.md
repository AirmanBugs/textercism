# xrc — Exercism workflow CLI

An interactive CLI for picking, solving, and submitting [Exercism](https://exercism.org)
exercises, synced through this git repo. Wraps the official `exercism` CLI for the
supported operations (download / test / submit) and reads the unofficial v2 website
API for per-exercise **status** (not-started / in-progress / completed / published / locked).

Replaces the old `<track>/start-exercise.sh` and `<track>/end-exercise.sh` scripts.

A Go program using [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the
interactive UI (arrow-key navigation, filtering).

## Requirements

- Go (to build) — `brew install go`.
- The official `exercism` CLI, configured (`exercism configure --token=...`). `xrc`
  reads the token, workspace, and API base from `~/.config/exercism/user.json`.
- `code` (VS Code) on `PATH` for the editor integration.

`xrc` finds the repo from its own binary location (`<repo>/tooling/xrc`), so it works
from any directory. Override with `XRC_REPO_ROOT` if needed.

## Build

```sh
cd tooling
go build -o xrc .       # produces ./xrc
```

Put `tooling/xrc` on your `PATH` (or symlink it) to run `xrc` from anywhere, e.g.:

```sh
ln -sf "$PWD/xrc" ~/.local/bin/xrc
```

## Usage

```sh
xrc                       # interactive: pick a track, then an exercise, then an action
xrc <track>               # interactive: jump straight to a track's exercises
xrc tracks                # list tracks with join state + progress
xrc list <track>          # list exercises with status badges
xrc start <track> <ex>    # download + open in VS Code (single window: README + solution)
xrc restart <track> <ex>  # re-download stubs (overwrites) + open
xrc open <track> <ex>     # open in VS Code (downloads first if missing)
xrc test <track> <ex>     # run the track's tests (mix test on elixir, else exercism test)
xrc submit <track> <ex>   # test, submit, then commit + push ("<track>: complete <ex>")
xrc web <track> <ex>      # open the exercise/solution page in the browser
```

### Status legend

```
● not started   ◐ in progress   ✓ completed   ★ published   🔒 locked
```

The `⬇` marker next to an exercise means it's downloaded into this repo locally.
Status itself always comes from the Exercism API, so it stays correct across devices
even before you download an exercise here.

## How it fits together

- **Status** is read-only from `https://exercism.org/api/v2` using the CLI token.
  This API is unofficial/undocumented, so `xrc` parses it leniently and never relies
  on undocumented *write* endpoints.
- **Start/continue** runs `git pull --ff-only` (requires a clean tree) so progress
  syncs from other devices before you work, then downloads into the repo and opens a
  **single-folder** VS Code window — one language server per window, avoiding the
  multi-root workspace crash.
- **Submit** runs tests, copies the exercise into the exercism workspace, runs
  `exercism submit`, then commits and pushes. A prior `"<track>: complete <ex>"`
  commit triggers a resubmit confirmation.
