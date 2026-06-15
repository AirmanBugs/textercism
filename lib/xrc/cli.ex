defmodule Xrc.CLI do
  @moduledoc """
  Entry point for the `xrc` escript.

  Usage:
    xrc                          interactive: pick a track, then an exercise
    xrc <track>                  interactive: exercises for a track
    xrc tracks                   list tracks with progress
    xrc list <track>             list exercises with status
    xrc start <track> <ex>       download + open in VS Code
    xrc restart <track> <ex>     re-download (overwrites stubs) + open
    xrc open <track> <ex>        open in VS Code (download if missing)
    xrc test <track> <ex>        run tests
    xrc submit <track> <ex>      test, submit, commit + push
    xrc web <track> <ex>         open exercise/solution page in browser
  """

  alias Xrc.{Actions, Config, TUI}

  def main(argv) do
    config = Config.load!()
    Xrc.Api.start_cache()
    dispatch(argv, config)
  rescue
    e in RuntimeError ->
      Owl.IO.puts([Owl.Data.tag("✘ ", :red), Exception.message(e)])
      System.halt(1)
  end

  defp dispatch([], config), do: TUI.run(config)
  defp dispatch(["tracks"], config), do: TUI.print_tracks(config)
  defp dispatch(["list", track], config), do: TUI.print_exercises(config, track)

  defp dispatch(["start", track, ex], config), do: Actions.start(config, track, ex, false)
  defp dispatch(["restart", track, ex], config), do: Actions.start(config, track, ex, true)
  defp dispatch(["open", track, ex], config), do: Actions.open(config, track, ex)
  defp dispatch(["test", track, ex], config), do: Actions.test(config, track, ex)
  defp dispatch(["submit", track, ex], config), do: Actions.submit(config, track, ex, &prompt/1)
  defp dispatch(["web", track, ex], config), do: Actions.web(config, track, ex)

  # `xrc <track>` -> interactive exercise picker for that track.
  defp dispatch([track], config), do: TUI.run(config, track)

  defp dispatch(_argv, _config) do
    IO.puts(@moduledoc)
    System.halt(1)
  end

  @doc "Yes/no prompt used for resubmit and commit confirmation in non-interactive paths."
  def prompt(question) do
    Owl.IO.confirm(message: question)
  end
end
