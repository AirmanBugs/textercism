defmodule Xrc.Actions do
  @moduledoc """
  High-level operations shared by the CLI and the interactive TUI. Each function
  takes a loaded `Config` and prints user-facing progress. These compose the
  lower-level modules (Api, ExercismCli, Git, VSCode, Local).
  """

  alias Xrc.{Api, Config, ExercismCli, Git, Local, Status, VSCode}

  @doc "Download (if needed) and open an exercise in VS Code. force? re-downloads stubs."
  def start(%Config{} = config, track, exercise, force? \\ false) do
    with :ok <- sync_before_work(config) do
      already = Local.downloaded?(config, track, exercise)

      cond do
        already and not force? ->
          info("Already downloaded; continuing.")
          open_in_vscode(config, track, exercise)

        true ->
          info("Downloading #{track}/#{exercise} ...")

          case ExercismCli.download(config, track, exercise) do
            {:ok, dir} ->
              ok("Downloaded to #{dir}")
              open_in_vscode(config, track, exercise)

            {:error, reason} ->
              error(reason)
          end
      end
    end
  end

  @doc "Open an already-downloaded exercise; download first if missing."
  def open(%Config{} = config, track, exercise) do
    if Local.downloaded?(config, track, exercise) do
      open_in_vscode(config, track, exercise)
    else
      info("Not downloaded yet — fetching first.")
      start(config, track, exercise, false)
    end
  end

  defp open_in_vscode(config, track, exercise) do
    case VSCode.open(config, track, exercise) do
      :ok ->
        ok("Opened #{track}/#{exercise} in VS Code.")

      {:error, :code_not_found} ->
        dir = Local.exercise_dir(config, track, exercise)
        info("`code` not on PATH. Open manually: #{dir}")
    end
  end

  @doc "Run the exercise's tests, streaming output."
  def test(%Config{} = config, track, exercise) do
    if Local.downloaded?(config, track, exercise) do
      info("Running tests for #{track}/#{exercise} ...")

      case ExercismCli.test(config, track, exercise) do
        :ok -> ok("Tests passed.")
        {:error, code} -> error("Tests failed (exit #{code}).")
      end
    else
      error("Exercise not downloaded. Start it first.")
    end
  end

  @doc """
  Test, submit, then commit + push. `confirm` is a 1-arg fun (prompt -> bool)
  used for the resubmit and commit confirmations; defaults to auto-yes for the
  non-interactive CLI path. Pass a real prompt fun from the TUI.
  """
  def submit(%Config{} = config, track, exercise, confirm \\ fn _ -> true end) do
    cond do
      not Local.downloaded?(config, track, exercise) ->
        error("Exercise not downloaded. Start it first.")

      Git.already_completed?(config, track, exercise) and
          not confirm.("This exercise was completed before. Resubmit?") ->
        info("Aborted.")

      true ->
        run_submit(config, track, exercise, confirm)
    end
  end

  defp run_submit(config, track, exercise, confirm) do
    info("Running tests before submit ...")

    case ExercismCli.test(config, track, exercise) do
      :ok ->
        ok("Tests passed. Submitting ...")

        case ExercismCli.submit(config, track, exercise) do
          {:ok, out} ->
            ok(String.trim(out))
            commit(config, track, exercise, confirm)

          {:error, reason} ->
            error(reason)
        end

      {:error, code} ->
        error("Tests failed (exit #{code}). Fix before submitting.")
    end
  end

  defp commit(config, track, exercise, confirm) do
    info("Committing to git ...")

    result =
      Git.commit_and_push(config, track, exercise, fn stat ->
        IO.puts("\nChanges to commit:\n#{stat}")
        confirm.("Commit and push these changes?")
      end)

    case result do
      :ok -> ok("Committed and pushed: #{track}: complete #{exercise}")
      :no_changes -> info("No file changes to commit (submission still sent).")
      :aborted -> info("Commit aborted; nothing pushed.")
      {:error, reason} -> error(reason)
    end
  end

  @doc "Open the exercise's web page in the browser."
  def web(%Config{} = config, track, exercise) do
    case Api.exercises(config, track) do
      {:ok, exercises} ->
        case Enum.find(exercises, &(&1.slug == exercise)) do
          nil -> error("Unknown exercise #{exercise} in #{track}.")
          ex -> open_url(ex.web_url)
        end

      {:error, reason} ->
        error("Could not fetch exercise: #{inspect(reason)}")
    end
  end

  @doc "Open an arbitrary URL in the default browser (macOS `open`, Linux `xdg-open`)."
  def open_url(url) do
    opener =
      cond do
        System.find_executable("open") -> "open"
        System.find_executable("xdg-open") -> "xdg-open"
        true -> nil
      end

    case opener do
      nil -> info("Open manually: #{url}")
      cmd -> System.cmd(cmd, [url]) && ok("Opened #{url}")
    end
  end

  @doc "git pull --ff-only before work, requiring a clean tree (like start-exercise.sh)."
  def sync_before_work(%Config{} = config) do
    cond do
      not Git.clean?(config) ->
        error("Working tree has uncommitted changes. Commit or stash first.")
        :error

      true ->
        case Git.pull_ff(config) do
          :ok ->
            :ok

          {:error, out} ->
            error("git pull --ff-only failed:\n#{out}")
            :error
        end
    end
  end

  # --- pretty output ---

  defp ok(msg), do: Owl.IO.puts([Owl.Data.tag("✔ ", :green), msg])
  defp info(msg), do: Owl.IO.puts([Owl.Data.tag("• ", :blue), msg])
  defp error(msg), do: Owl.IO.puts([Owl.Data.tag("✘ ", :red), msg])

  @doc false
  def status_badge(status), do: Status.badge(status)
end
