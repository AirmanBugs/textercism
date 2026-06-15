# textercism — a text UI for Exercism

An interactive CLI for picking, solving, and submitting [Exercism](https://exercism.org)
exercises. Wraps the official `exercism` CLI for the supported operations
(download / test / submit) and reads the unofficial v2 website API for per-exercise
**status** (not started / started / in progress / completed / published / locked).

A Go program using [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the
interactive UI (arrow-key navigation, filtering).

## Requirements

- Go (to build) — `brew install go`.
- The official `exercism` CLI, configured (`exercism configure --token=...`).
  textercism reuses the **token** and **workspace** from `~/.config/exercism/user.json` —
  no separate config needed.
- `code` (VS Code) on `PATH` for the editor integration (optional).

Exercises live in the Exercism CLI **workspace** (`~/Exercism` by default), so
textercism works from any directory.

## Build

```sh
go build -o textercism .       # produces ./textercism
```

Put it on your `PATH` (or symlink it) to run from anywhere, e.g.:

```sh
ln -sf "$PWD/textercism" ~/.local/bin/textercism
```

## Usage

```sh
textercism                       # interactive: pick a track, then an exercise, then an action
textercism <track>               # interactive: jump straight to a track's exercises
textercism tracks                # list tracks with join state + progress
textercism list <track>          # list exercises with status badges
textercism start <track> <ex>    # download + open solution in VS Code
textercism restart <track> <ex>  # re-download stub (overwrites) + open
textercism open <track> <ex>     # open solution in VS Code (downloads/syncs if missing)
textercism read <track> <ex>     # render the exercise instructions in the terminal
textercism test <track> <ex>     # run the track's tests (mix test on elixir, else exercism test)
textercism submit <track> <ex>   # test, then submit to Exercism
textercism pause <track> <ex>    # save draft to your sync backend (when configured)
textercism web <track> <ex>      # open the exercise/solution page in the browser
```

### Status legend

```
● not started   ◌ started (server)   ◔ started   ◐ in progress   ✓ completed   ★ published   🔒 locked
```

Status combines two sources:

- **Exercism API** (the server) — knows whether you've *started* (a solution record
  exists), *completed*, or *published*. But Exercism only stores code you've
  **submitted**, not in-progress drafts.
- **Local disk** — whether the exercise is downloaded in the workspace (`⬇`) and
  whether its solution has been **edited** away from the pristine stub.

Merged, that gives:

- `◌ started (server)` — Exercism says you started it, but there's nothing local to
  continue (e.g. started on another machine). **Continue** downloads the stub (or
  pulls your draft from a sync backend, if configured).
- `◔ started` — downloaded here, stub untouched.
- `◐ in progress` — downloaded here with real edits (a partial solution).

## How it fits together

- **Storage** is the Exercism CLI workspace (`~/Exercism`); exercises download there
  and submit happens in place. Completed work lives on **Exercism** (the source of
  truth) — textercism keeps no separate copy and re-fetches as needed.
- **Status** is read-only from `https://exercism.org/api/v2` using the CLI token.
  This API is unofficial/undocumented, so textercism parses it leniently and never
  relies on undocumented *write* endpoints.
- **Start / continue** opens a **single-folder** VS Code window (one language server,
  avoiding the multi-root workspace crash).
- **Instructions** render in the terminal via [Glamour](https://github.com/charmbracelet/glamour):
  selecting an exercise shows its rendered instructions in a pane **beside the action
  list** (stacked on narrow terminals). Scroll the pane with the trackpad/wheel, or
  press **Tab** to focus it and use ↑/↓. Not-yet-downloaded
  exercises are fetched automatically so the full instructions appear. `textercism
  read <track> <ex>` also prints the rendered README to stdout. The `web` command
  opens the exercise on exercism.org in a browser.
- **Actions run in the TUI.** Start/open/restart/pause run in the background with a
  status line; Submit runs tests then submits.
- **Run tests** parses the `mix test` output into a clean result in the right pane —
  a pass/fail banner plus each failed test's name, location, and assertion
  (code, left, right) — with compile warnings and stacktraces stripped. Press `r`
  for the raw output, `i` to return to instructions.
- **Submit** runs the tests, then `exercism submit` in place.
- **Drafts** (in-progress, unsubmitted code) are the only thing that needs syncing,
  since Exercism doesn't store them. Cross-device sync is handled by a pluggable
  **sync backend** behind a small interface (`internal/sync`). The default is
  **local-only** (no cross-device sync; the "Pause & sync" action is hidden). Future
  backends — a synced folder (Dropbox/iCloud/a Pi mount), ssh/scp to a personal
  server, or git — slot in without touching the rest of the tool.
