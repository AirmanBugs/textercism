defmodule Xrc.VSCode do
  @moduledoc """
  Opens an exercise in VS Code as a single-folder window (one ElixirLS server,
  avoiding the multi-root workspace crash). Opens the README and the solution
  file in side-by-side panes, with the README shown as a rendered preview.
  """

  alias Xrc.{Config, Local}

  @doc """
  Open the exercise window. Returns :ok, or {:error, reason} if `code` isn't
  available (in which case the caller can print the dir for manual opening).
  """
  def open(%Config{} = config, track, exercise) do
    dir = Local.exercise_dir(config, track, exercise)

    case System.find_executable("code") do
      nil ->
        {:error, :code_not_found}

      code ->
        readme = Local.readme(config, track, exercise)
        solution = first_solution_path(config, track, exercise)

        # Open the folder (new window), then the files as tabs/panes.
        files = Enum.reject([solution, readme], &is_nil/1)
        System.cmd(code, ["--new-window", dir | files], stderr_to_stdout: true)

        # Show the README as a rendered preview beside the solution.
        if readme do
          System.cmd(code, ["--reuse-window", "--command", "markdown.showPreviewToSide"],
            stderr_to_stdout: true
          )
        end

        :ok
    end
  end

  defp first_solution_path(config, track, exercise) do
    dir = Local.exercise_dir(config, track, exercise)

    case Local.solution_files(config, track, exercise) do
      [rel | _] -> Path.join(dir, rel)
      [] -> nil
    end
  end
end
