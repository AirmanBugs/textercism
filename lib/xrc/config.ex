defmodule Xrc.Config do
  @moduledoc """
  Reads the Exercism CLI configuration (`user.json`) and resolves the local
  git repo root. The CLI's API token also authenticates against the unofficial
  v2 website API, so it's the single source of credentials for `xrc`.
  """

  @v2_base "https://exercism.org/api/v2"

  defstruct [:token, :workspace, :apibaseurl, :repo_root]

  @type t :: %__MODULE__{
          token: String.t(),
          workspace: String.t(),
          apibaseurl: String.t(),
          repo_root: String.t()
        }

  @doc "The fixed base URL for the unofficial v2 website API (not the CLI's v1 base)."
  def v2_base, do: @v2_base

  @doc """
  Load configuration, or raise with a friendly message if the CLI isn't set up.
  """
  @spec load!() :: t()
  def load! do
    path = user_json_path()

    unless File.exists?(path) do
      raise """
      Exercism CLI config not found at #{path}.
      Run `exercism configure --token=<your-token>` first.
      """
    end

    json = path |> File.read!() |> Jason.decode!()

    %__MODULE__{
      token: fetch!(json, "token", path),
      workspace: fetch!(json, "workspace", path),
      apibaseurl: Map.get(json, "apibaseurl", "https://api.exercism.org/v1"),
      repo_root: repo_root()
    }
  end

  defp fetch!(json, key, path) do
    case Map.get(json, key) do
      nil -> raise "Missing #{inspect(key)} in #{path}. Re-run `exercism configure`."
      "" -> raise "Empty #{inspect(key)} in #{path}. Re-run `exercism configure`."
      value -> value
    end
  end

  @doc """
  Path to the Exercism CLI `user.json`, honoring the same env vars the CLI does:
  `EXERCISM_CONFIG_HOME`, then `XDG_CONFIG_HOME/exercism`, then `~/.config/exercism`.
  """
  def user_json_path do
    dir =
      cond do
        dir = System.get_env("EXERCISM_CONFIG_HOME") -> dir
        dir = System.get_env("XDG_CONFIG_HOME") -> Path.join(dir, "exercism")
        true -> Path.join(System.user_home!(), ".config/exercism")
      end

    Path.join(dir, "user.json")
  end

  @doc """
  The repo root holding the per-track exercise folders. Resolved from git when
  available, otherwise from the current working directory.
  """
  def repo_root do
    cond do
      dir = System.get_env("XRC_REPO_ROOT") -> dir
      dir = repo_root_from_escript() -> dir
      dir = repo_root_from_git() -> dir
      true -> File.cwd!()
    end
  end

  # The escript lives at <repo>/tooling/xrc; resolve <repo> from its real path so
  # `xrc` works from any directory (e.g. via a symlink on PATH).
  defp repo_root_from_escript do
    with name when is_list(name) <- :escript.script_name(),
         path <- List.to_string(name),
         real <- resolve_symlink(path),
         "tooling" <- Path.basename(Path.dirname(real)) do
      real |> Path.dirname() |> Path.dirname()
    else
      _ -> nil
    end
  rescue
    _ -> nil
  end

  defp resolve_symlink(path) do
    case File.read_link(path) do
      {:ok, target} -> Path.expand(target, Path.dirname(path))
      _ -> Path.expand(path)
    end
  end

  defp repo_root_from_git do
    case System.cmd("git", ["rev-parse", "--show-toplevel"], stderr_to_stdout: true) do
      {out, 0} -> String.trim(out)
      _ -> nil
    end
  end
end
