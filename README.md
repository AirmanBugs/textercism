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
xrc start <track> <ex>    # download + open solution in VS Code, instructions in browser
xrc restart <track> <ex>  # re-download stubs (overwrites) + open
xrc open <track> <ex>     # open solution in VS Code + instructions in browser (downloads if missing)
xrc test <track> <ex>     # run the track's tests (mix test on elixir, else exercism test)
xrc submit <track> <ex>   # test, submit, then commit + push ("<track>: complete <ex>")
xrc pause <track> <ex>    # commit work-in-progress + push (sync drafts across devices)
xrc web <track> <ex>      # open the exercise/solution page in the browser
```

### Status legend

```
● not started   ◌ started (server)   ◔ started   ◐ in progress   ✓ completed   ★ published   🔒 locked
```

Status combines two sources:

- **Exercism API** (the server) — knows whether you've *started* (a solution record
  exists), *completed*, or *published*. But Exercism only stores code you've
  **submitted**, not drafts.
- **Local disk** — whether the exercise is downloaded here (`⬇`) and whether its
  solution has been **edited** away from the pristine stub.

Merged, that gives:

- `◌ started (server)` — Exercism says you started it, but there's nothing local to
  continue (e.g. started on another machine and not yet synced). **Continue** will
  download the stub.
- `◔ started` — downloaded here, stub untouched.
- `◐ in progress` — downloaded here with real edits (a partial solution).

To carry a partial solution between machines, use **Pause & sync**: it commits a
`"<track>: wip <ex>"` snapshot and pushes. On another device, **Continue** pulls it
back. When you finally **Submit**, any `wip` commits are squashed into the single
`"<track>: complete <ex>"` commit.

## How it fits together

- **Status** is read-only from `https://exercism.org/api/v2` using the CLI token.
  This API is unofficial/undocumented, so `xrc` parses it leniently and never relies
  on undocumented *write* endpoints.
- **Start/continue** runs `git pull --ff-only` (requires a clean tree) so progress
  syncs from other devices before you work, then downloads into the repo and opens a
  **single-folder** VS Code window — one language server per window, avoiding the
  multi-root workspace crash. The exercise's instructions open in the browser
  (exercism.org), where they render properly and can sit beside the editor. VS Code
  1.124's `code` CLI has no `--command` flag, so xrc can't arrange a README pane
  inside the editor — the browser is the reliable place for instructions.
- **Submit** runs tests, copies the exercise into the exercism workspace, runs
  `exercism submit`, then commits and pushes. A prior `"<track>: complete <ex>"`
  commit triggers a resubmit confirmation.
