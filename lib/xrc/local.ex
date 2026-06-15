defmodule Xrc.Local do
  @moduledoc """
  Local on-disk state in the git repo. Exercises live at
  `<repo_root>/<track>/<exercise>/`. The API is the source of truth for status;
  this module answers "is it downloaded here, and what are its files".
  """

  alias Xrc.Config

  @doc "Absolute path to an exercise directory in the repo."
  def exercise_dir(%Config{repo_root: root}, track, exercise) do
    Path.join([root, track, exercise])
  end

  @doc "True if the exercise has been downloaded into the repo (has a .exercism dir)."
  def downloaded?(%Config{} = config, track, exercise) do
    config
    |> exercise_dir(track, exercise)
    |> Path.join(".exercism")
    |> File.dir?()
  end

  @doc """
  Solution file paths (relative to the exercise dir) from `.exercism/config.json`.
  Falls back to an empty list if the metadata is missing/unreadable.
  """
  def solution_files(%Config{} = config, track, exercise) do
    config
    |> exercise_dir(track, exercise)
    |> Path.join(".exercism/config.json")
    |> read_json()
    |> case do
      {:ok, %{"files" => %{"solution" => files}}} when is_list(files) -> files
      _ -> []
    end
  end

  @doc "Path to the exercise's README.md, if present."
  def readme(%Config{} = config, track, exercise) do
    path = config |> exercise_dir(track, exercise) |> Path.join("README.md")
    if File.exists?(path), do: path, else: nil
  end

  defp read_json(path) do
    with {:ok, raw} <- File.read(path),
         {:ok, json} <- Jason.decode(raw) do
      {:ok, json}
    end
  end
end
