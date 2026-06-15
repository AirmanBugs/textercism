// Command xrc is an interactive CLI for picking, solving, and submitting
// Exercism exercises, synced through this git repo.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/AirmanBugs/exercism/xrc/internal/actions"
	"github.com/AirmanBugs/exercism/xrc/internal/config"
	"github.com/AirmanBugs/exercism/xrc/internal/exercism"
	"github.com/AirmanBugs/exercism/xrc/internal/tui"
)

const usage = `xrc — Exercism workflow CLI

Usage:
  xrc                       interactive: pick a track, then an exercise, then an action
  xrc <track>               interactive: jump to a track's exercises
  xrc tracks                list tracks with join state + progress
  xrc list <track>          list exercises with status
  xrc start <track> <ex>    download + open in VS Code
  xrc restart <track> <ex>  re-download stubs (overwrites) + open
  xrc open <track> <ex>     open in VS Code (downloads first if missing)
  xrc test <track> <ex>     run the track's tests
  xrc submit <track> <ex>   test, submit, commit + push
  xrc pause <track> <ex>    commit work-in-progress + push (sync across devices)
  xrc web <track> <ex>      open the exercise/solution page in the browser
`

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "✘ "+err.Error())
		os.Exit(1)
	}

	args := os.Args[1:]
	switch {
	case len(args) == 0:
		runTUI(cfg, "")

	case args[0] == "tracks" && len(args) == 1:
		printTracks(cfg)
	case args[0] == "list" && len(args) == 2:
		printExercises(cfg, args[1])

	case args[0] == "start" && len(args) == 3:
		actions.Start(cfg, args[1], args[2], false)
	case args[0] == "restart" && len(args) == 3:
		actions.Start(cfg, args[1], args[2], true)
	case args[0] == "open" && len(args) == 3:
		actions.Open(cfg, args[1], args[2])
	case args[0] == "test" && len(args) == 3:
		actions.Test(cfg, args[1], args[2])
	case args[0] == "submit" && len(args) == 3:
		actions.Submit(cfg, args[1], args[2], promptConfirm)
	case args[0] == "pause" && len(args) == 3:
		actions.Pause(cfg, args[1], args[2])
	case args[0] == "web" && len(args) == 3:
		actions.Web(cfg, args[1], args[2])

	// `xrc <track>` -> interactive exercises for that track.
	case len(args) == 1:
		runTUI(cfg, args[0])

	default:
		fmt.Print(usage)
		os.Exit(1)
	}
}

// runTUI launches the interactive UI, then performs the chosen action (outside
// the alt-screen so its output goes to the real terminal).
func runTUI(cfg *config.Config, startTrack string) {
	action, err := tui.Run(cfg, startTrack)
	if err != nil {
		fmt.Fprintln(os.Stderr, "✘ "+err.Error())
		os.Exit(1)
	}
	perform(cfg, action)
}

func perform(cfg *config.Config, a tui.Action) {
	switch a.Kind {
	case tui.ActionStart:
		actions.Start(cfg, a.Track, a.Exercise, false)
	case tui.ActionRestart:
		actions.Start(cfg, a.Track, a.Exercise, true)
	case tui.ActionOpen:
		actions.Open(cfg, a.Track, a.Exercise)
	case tui.ActionTest:
		actions.Test(cfg, a.Track, a.Exercise)
	case tui.ActionSubmit:
		actions.Submit(cfg, a.Track, a.Exercise, promptConfirm)
	case tui.ActionPause:
		actions.Pause(cfg, a.Track, a.Exercise)
	case tui.ActionWeb:
		actions.Web(cfg, a.Track, a.Exercise)
	case tui.ActionNone:
		// user quit; nothing to do
	}
}

func printTracks(cfg *config.Config) {
	client := exercism.NewClient(cfg)
	tracks, err := client.Tracks()
	if err != nil {
		fmt.Fprintln(os.Stderr, "✘ "+err.Error())
		os.Exit(1)
	}
	for _, t := range tracks {
		marker := "·"
		if t.IsJoined {
			marker = "✔"
		}
		fmt.Printf("%s %-22s %d/%d\n", marker, t.Slug, t.NumCompletedExercises, t.NumExercises)
	}
}

func printExercises(cfg *config.Config, track string) {
	client := exercism.NewClient(cfg)
	exs, err := client.Exercises(track)
	if err != nil {
		fmt.Fprintln(os.Stderr, "✘ "+err.Error())
		os.Exit(1)
	}
	fmt.Println(exercism.Legend())
	for _, e := range exs {
		state := exercism.LocalStateOf(cfg, track, e.Slug)
		display := exercism.Display(e.Status, state)
		local := " "
		if state != exercism.NotOnDisk {
			local = "⬇"
		}
		diff := e.Difficulty
		if diff == "" {
			diff = "—"
		}
		rec := ""
		if e.IsRecommended {
			rec = " ★rec"
		}
		fmt.Printf("%s %s %-28s [%s]%s\n", display.Badge(), local, e.Title, diff, rec)
	}
}

func promptConfirm(question string) bool {
	fmt.Print(question + " [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}
