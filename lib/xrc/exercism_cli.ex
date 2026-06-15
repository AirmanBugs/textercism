defmodule Xrc.ExercismCli do
  @moduledoc """
  Wraps the official `exercism` CLI for the stable, supported operations:
  download, test, submit. Mirrors the move/copy logic that the previous
  start-exercise.sh / end-exercise.sh used so files stay in the git repo.
  """

  alias Xrc.{Config, Local}

  @doc """
  Download an exercise via the CLI, then move it from the exercism workspace
  into the repo at `<repo>/<track>/<exercise>`. `--force` re-downloads stubs.
  Returns {:ok, dir} | {:error, reason}.
  """
  def download(%Config{} = config, track, exercise) do
    args = ["download", "--track=#{track}", "--exercise=#{exercise}", "--force"]

    case System.cmd("exercism", args, stderr_to_stdout: true) do
      {out, 0} ->
        move_into_repo(config, track, exercise, out)

      {out, code} ->
        {:error, "exercism download failed (#{code}):\n#{out}"}
    end
  end

  # The CLI prints "Downloaded to\n<path>". Move that into the repo dir.
  defp move_into_repo(config, track, exercise, output) do
    target = Local.exercise_dir(config, track, exercise)

    downloaded_path =
      output
      |> String.split("\n")
      |> Enum.drop_while(&(not String.contains?(&1, "Downloaded to")))
      |> Enum.drop(1)
      |> List.first()
      |> case do
        nil -> nil
        line -> String.trim(line)
      end

    cond do
      File.dir?(target) and (is_nil(downloaded_path) or downloaded_path == target) ->
        # Already in place (CLI workspace == repo, or nothing to move).
        {:ok, target}

      downloaded_path && File.dir?(downloaded_path) ->
        File.mkdir_p!(Path.dirname(target))
        File.rm_rf!(target)
        File.rename(downloaded_path, target)
        {:ok, target}

      File.dir?(target) ->
        {:ok, target}

      true ->
        {:error, "Could not locate downloaded exercise from CLI output:\n#{output}"}
    end
  end

  @doc """
  Run the exercise's tests, streaming output to the terminal. Elixir uses
  `mix test`; other tracks use `exercism test` (the track's configured runner).
  Returns :ok | {:error, exit_code}.
  """
  def test(%Config{} = config, track, exercise) do
    dir = Local.exercise_dir(config, track, exercise)
    {cmd, args} = test_command(track)

    status = run_streamed(cmd, args, dir)

    if status == 0, do: :ok, else: {:error, status}
  end

  defp test_command("elixir"), do: {"mix", ["test"]}
  defp test_command(_other), do: {"exercism", ["test"]}

  @doc """
  Submit the exercise's solution files. Copies the exercise into the exercism
  workspace (as end-exercise.sh did) and runs `exercism submit` from there, then
  removes the copy. Solution files come from `.exercism/config.json`.
  """
  def submit(%Config{} = config, track, exercise) do
    repo_dir = Local.exercise_dir(config, track, exercise)
    files = Local.solution_files(config, track, exercise)

    cond do
      not File.dir?(repo_dir) ->
        {:error, "Exercise not downloaded: #{repo_dir}"}

      files == [] ->
        {:error, "No solution files listed in .exercism/config.json for #{exercise}"}

      true ->
        do_submit(config, track, exercise, repo_dir, files)
    end
  end

  defp do_submit(config, track, exercise, repo_dir, files) do
    ws_dir = Path.join([config.workspace, track, exercise])
    File.mkdir_p!(Path.dirname(ws_dir))
    File.rm_rf!(ws_dir)
    File.cp_r!(repo_dir, ws_dir)

    result = System.cmd("exercism", ["submit" | files], cd: ws_dir, stderr_to_stdout: true)

    File.rm_rf!(ws_dir)

    case result do
      {out, 0} -> {:ok, out}
      {out, code} -> {:error, "exercism submit failed (#{code}):\n#{out}"}
    end
  end

  # Run a command in `dir`, writing combined output to stdout as it arrives.
  defp run_streamed(cmd, args, dir) do
    port =
      Port.open({:spawn_executable, System.find_executable(cmd)}, [
        :binary,
        :exit_status,
        :stderr_to_stdout,
        {:args, args},
        {:cd, dir},
        :hide
      ])

    stream_loop(port)
  end

  defp stream_loop(port) do
    receive do
      {^port, {:data, data}} ->
        IO.write(data)
        stream_loop(port)

      {^port, {:exit_status, status}} ->
        status
    end
  end
end
