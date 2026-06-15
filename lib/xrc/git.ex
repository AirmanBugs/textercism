defmodule Xrc.Git do
  @moduledoc """
  Git helpers mirroring the proven flow from start-exercise.sh / end-exercise.sh:
  clean-tree check + `pull --ff-only` before work, and stage/commit/push with a
  `"<track>: complete <exercise>"` message after a successful submit. A
  duplicate-submit guard checks the log for a prior completion commit.
  """

  alias Xrc.Config

  @doc "True if the working tree (tracked files) is clean."
  def clean?(%Config{repo_root: root}) do
    git_ok?(root, ["diff", "--quiet"]) and git_ok?(root, ["diff", "--cached", "--quiet"])
  end

  @doc "Fast-forward pull. Returns :ok | {:error, output}."
  def pull_ff(%Config{repo_root: root}) do
    case git(root, ["pull", "--ff-only"]) do
      {_out, 0} -> :ok
      {out, _} -> {:error, out}
    end
  end

  @doc "True if a prior `\"<track>: complete <exercise>\"` commit exists."
  def already_completed?(%Config{repo_root: root}, track, exercise) do
    {out, _} = git(root, ["log", "--oneline", "--grep=#{track}: complete #{exercise}"])
    String.trim(out) != ""
  end

  @doc """
  Stage the exercise dir, show the stat diff, commit `"<track>: complete <exercise>"`,
  and push. `confirm_fun` is called with the staged stat text and must return a
  boolean. Returns :ok | {:error, reason} | :no_changes | :aborted.
  """
  def commit_and_push(%Config{repo_root: root}, track, exercise, confirm_fun) do
    rel = Path.join(track, exercise)

    if not has_changes?(root, rel) do
      :no_changes
    else
      git(root, ["add", rel <> "/"])
      {stat, _} = git(root, ["diff", "--cached", "--stat"])

      if confirm_fun.(stat) do
        case git(root, ["commit", "-m", "#{track}: complete #{exercise}"]) do
          {_out, 0} ->
            case git(root, ["push"]) do
              {_out, 0} -> :ok
              {out, _} -> {:error, "push failed:\n#{out}"}
            end

          {out, _} ->
            {:error, "commit failed:\n#{out}"}
        end
      else
        git(root, ["reset", "HEAD"])
        :aborted
      end
    end
  end

  defp has_changes?(root, rel) do
    {out, _} = git(root, ["status", "--porcelain", "--", rel])
    String.trim(out) != ""
  end

  defp git_ok?(root, args) do
    {_out, code} = git(root, args)
    code == 0
  end

  defp git(root, args) do
    System.cmd("git", ["-C", root | args], stderr_to_stdout: true)
  end
end
