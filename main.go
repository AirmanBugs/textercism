// Command textercism is an interactive CLI for picking, solving, and submitting
// Exercism exercises.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/AirmanBugs/textercism/internal/actions"
	"github.com/AirmanBugs/textercism/internal/config"
	"github.com/AirmanBugs/textercism/internal/exercism"
	"github.com/AirmanBugs/textercism/internal/sync"
	"github.com/AirmanBugs/textercism/internal/tui"
)

const usage = `textercism — a text UI for Exercism

Usage:
  textercism                       interactive: pick a track, then an exercise, then an action
  textercism <track>               interactive: jump to a track's exercises
  textercism tracks                list tracks with join state + progress
  textercism list <track>          list exercises with status
  textercism start <track> <ex>    download + open solution in VS Code, instructions in browser
  textercism restart <track> <ex>  re-download stub (overwrites) + open
  textercism open <track> <ex>     open solution + instructions (downloads/syncs if missing)
  textercism test <track> <ex>     run the track's tests
  textercism submit <track> <ex>   test, then submit to Exercism
  textercism pause <track> <ex>    save draft to your sync backend (when configured)
  textercism web <track> <ex>      open the exercise/solution page in the browser
`

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "✘ "+err.Error())
		os.Exit(1)
	}

	// Local-only backend for now; future sync backends are selected here.
	backend := sync.NewLocal(cfg)

	args := os.Args[1:]
	switch {
	case len(args) == 0:
		runTUI(cfg, backend, "")

	case args[0] == "tracks" && len(args) == 1:
		printTracks(cfg)
	case args[0] == "list" && len(args) == 2:
		printExercises(cfg, args[1])

	case args[0] == "start" && len(args) == 3:
		actions.Start(cfg, backend, args[1], args[2], false)
	case args[0] == "restart" && len(args) == 3:
		actions.Start(cfg, backend, args[1], args[2], true)
	case args[0] == "open" && len(args) == 3:
		actions.Open(cfg, backend, args[1], args[2])
	case args[0] == "test" && len(args) == 3:
		actions.Test(cfg, args[1], args[2])
	case args[0] == "submit" && len(args) == 3:
		actions.Submit(cfg, args[1], args[2], promptConfirm)
	case args[0] == "pause" && len(args) == 3:
		actions.Pause(cfg, backend, args[1], args[2])
	case args[0] == "web" && len(args) == 3:
		actions.Web(cfg, args[1], args[2])

	// `textercism <track>` -> interactive exercises for that track.
	case len(args) == 1:
		runTUI(cfg, backend, args[0])

	default:
		fmt.Print(usage)
		os.Exit(1)
	}
}

// runTUI launches the interactive UI, then performs the chosen action (outside
// the alt-screen so its output goes to the real terminal).
func runTUI(cfg *config.Config, backend sync.Backend, startTrack string) {
	action, err := tui.Run(cfg, startTrack, backend.SyncsAcrossDevices())
	if err != nil {
		fmt.Fprintln(os.Stderr, "✘ "+err.Error())
		os.Exit(1)
	}
	perform(cfg, backend, action)
}

func perform(cfg *config.Config, backend sync.Backend, a tui.Action) {
	switch a.Kind {
	case tui.ActionStart:
		actions.Start(cfg, backend, a.Track, a.Exercise, false)
	case tui.ActionRestart:
		actions.Start(cfg, backend, a.Track, a.Exercise, true)
	case tui.ActionOpen:
		actions.Open(cfg, backend, a.Track, a.Exercise)
	case tui.ActionTest:
		actions.Test(cfg, a.Track, a.Exercise)
	case tui.ActionSubmit:
		actions.Submit(cfg, a.Track, a.Exercise, promptConfirm)
	case tui.ActionPause:
		actions.Pause(cfg, backend, a.Track, a.Exercise)
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
