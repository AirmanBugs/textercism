# textercism — re-architecture design

> Status: **draft for review**. Nothing here is built yet. The current tool works
> but couples the CLI to a personal git repo; this doc defines the standalone,
> publishable architecture.

## Goals

1. **Decouple the tool from storage.** The CLI becomes a standalone, publishable
   program (Homebrew). It must not assume it lives inside the user's solutions repo.
2. **Right-size storage.** The only thing that genuinely needs persistence is
   **in-progress drafts** (latest state, no history). Completed exercises live on
   Exercism and are re-fetched — storing them ourselves is redundant.
3. **Local-first, sync optional.** Works fully offline on one machine with zero
   setup beyond the Exercism token. Cross-device sync is an opt-in add-on behind a
   small interface.

## Non-goals

- Version history / multiple saved attempts per exercise (decided: "latest draft
  only"). No VCS semantics — last write wins.
- Storing completed solutions (Exercism has them).
- Mentoring, publishing, or other website-only features.

## What changes from today

| Today | New |
|---|---|
| Tool lives at `<solutions-repo>/tooling/` | Standalone repo, installed on PATH |
| `repoRoot` derived from binary location | Drafts dir = Exercism CLI **workspace** (`user.json`) |
| Storage = the git repo the tool sits in | Storage = the workspace + optional sync backend |
| WIP commit + squash-on-complete (git history) | Dropped. Sync is "save latest snapshot" |
| `git pull --ff-only` before work | `backend.Pull()` (no-op for local) |
| Submit copies exercise into workspace | Submit in place (already in workspace) |

## Storage model

The drafts dir **is the Exercism CLI's own workspace** (read from `user.json`'s
`workspace`, default `~/Exercism`). This means downloads land where `exercism`
already puts them, and **submit happens in place** — no more copy-into-workspace
step (today's `Submit` copies the exercise into the workspace just to run
`exercism submit`; that goes away).

```
<workspace>/<track>/<exercise>/      # e.g. ~/Exercism/elixir/two-fer/
   lib/two_fer.ex                    # the solution (the draft)
   test/...                          # from the exercise download
   README.md, .exercism/, ...
```

- **Latest draft only.** No history kept by textercism.
- **Completed exercises**: kept locally only while working; safe to delete after
  completion since Exercism is the source of truth (re-fetched on demand).
- Local status detection (downloaded? edited vs. stub?) stays as-is — it already
  reads the disk and compares against the captured stub, independent of git.

## The SyncBackend interface

The single seam between the tool and "where drafts live for cross-device use".
Because we only need the latest draft, it stays tiny — no merge/conflict logic.

```go
// SyncBackend persists in-progress drafts so they can move between devices.
// Semantics: last write wins. Implementations need not version anything.
type SyncBackend interface {
    // Name is shown in config/status (e.g. "local", "git", "s3").
    Name() string

    // Push saves the exercise's current draft (the files under its dir).
    Push(track, exercise string, dir string) error

    // Pull fetches the latest stored draft into dir (overwrites). Returns
    // ErrNoDraft if none exists remotely.
    Pull(track, exercise string, dir string) error

    // List returns the track/exercise pairs that have stored drafts.
    List() ([]DraftRef, error)
}

type DraftRef struct{ Track, Exercise string }
```

Mapped onto the existing actions:

- **Start / Continue**: `backend.Pull(...)` (recover a draft from another device)
  before opening; if nothing local and nothing remote, `exercism download` the stub.
- **Pause & sync** → `backend.Push(...)`. With only `LocalBackend` shipping, this is
  a **no-op** initially, so the action is **hidden/disabled until a sync backend is
  configured** (no point offering "sync" that does nothing). It returns when sync lands.
- **Submit** → `exercism submit` in place. **No squash, no git commit** — drafts
  aren't versioned.

### Backends

- **LocalBackend (default, always available).** Drafts live in the workspace;
  `Push`/`Pull` are no-ops (same machine), `List` walks the dir. Zero setup.
  **This is all we ship initially.**
- **Future sync backends (deferred — not built now).** Behind the same interface,
  user-configured. Likely candidates the user has in mind:
  - **Synced folder** — point a backend at a folder something else syncs (Dropbox,
    iCloud, a Raspberry Pi mount). textercism writes plain files; the OS/service syncs.
  - **ssh/scp to a personal server** (e.g. the user's Raspberry Pi).
  - **git / GitHub** for those who want it.
  These are mutually compatible; config selects one. None affect actions/TUI.

## Config

Both the Exercism **token** and the **workspace** are reused from the existing
`exercism` CLI config (`user.json`) — we don't ask for them twice, and initially
textercism needs **no config file of its own**.

A textercism config file (`~/.config/textercism/config.toml`) is introduced only
once sync backends arrive, to select/configure one:

```toml
# (future) — not needed for the local-only initial release
sync = "local"                   # "local" = no cross-device sync
# [folder]  dir = "~/Dropbox/exercism-drafts"
# [ssh]     host = "pi"; path = "/srv/exercism-drafts"
# [git]     remote = "git@github.com:me/exercism-drafts.git"
```

Resolution for token + workspace: env override (`TEXTERCISM_*`) → textercism
config (future) → the official CLI's `user.json` (the default path).

## Repo split / migration

1. **Refactor in place first** (this codebase), so the extracted code is already
   correct for standalone use:
   - Replace `config.repoRoot()` (binary-location) with `config.Workspace` (already
     read from `user.json`); re-point all `ExerciseDir(...)` at the workspace.
   - Remove `WipCommitAndPush`, `SquashWipInto`, `countTrailingWip`,
     `AlreadyCompleted`, and the git pull/commit/push flow; introduce
     `SyncBackend` + `LocalBackend` (the only backend now).
   - Drop the copy-into-workspace step in `Submit` (download already lands in the
     workspace; submit in place).
   - Module path / binary become `textercism`.
2. **Extract** the refactored `tooling/` into a fresh `textercism` repo (Go module,
   README, Homebrew formula later).
3. **Abandon** the old `exercism` repo (left as-is, unused — user's choice).
4. **(Later)** implement a sync backend; publish + Homebrew.

## What gets deleted

- `internal/exercism/git.go` WIP/squash logic and its tests
  (`TestSquashWipIntoComplete`, `TestWipCommitNoChanges`).
- Binary-location repo-root resolution in `config.go`.
- `Pause`'s git-commit implementation (replaced by `backend.Push`).

`AlreadyCompleted` (git-log grep) also goes — completion state comes from the
Exercism API, not git history.

## Naming

**`textercism`** — "a text-based UI for Exercism." Provisional; the user may
change it. It only touches the binary output, Go module path, README, and Homebrew
formula, so a later rename is cheap and localized.

## Resolved decisions

- **Ship local-only first**; sync backends deferred behind the interface.
- **Future sync candidates** (user's): synced folder (Dropbox/iCloud/Pi mount),
  ssh/scp to a personal server (Raspberry Pi), git/GitHub — user-configured, all
  compatible with the one interface.
- **Drafts dir = the Exercism CLI workspace** (`user.json` `workspace`), which lets
  us drop the copy-on-submit step.
- **Keep the generic all-tracks design** (already present, cheap).

## Open questions (small; can decide during implementation)

1. **After submit:** keep the local draft, or remove it (it's now on Exercism and
   re-fetchable)? Leaning **keep** — re-downloading on every glance is wasteful, and
   disk is cheap; offer a `clean` command later.
2. **`status`/listing source for completed-but-not-local exercises:** unchanged —
   the Exercism API (`completed`/`published`) drives the badge; local presence only
   refines the in-between states. (No `AlreadyCompleted` git grep needed.)
