defmodule Xrc.Api do
  @moduledoc """
  Read-only client for Exercism's unofficial v2 website API. The CLI's API token
  (from `user.json`) authenticates as a Bearer token. These endpoints are
  undocumented and may change, so callers should treat fields as best-effort.

  Responses are cached for the lifetime of the OS process (the escript run) to be
  polite to the undocumented API. Pass `refresh: true` to bypass the cache.
  """

  alias Xrc.{Config, Status}

  @cache __MODULE__.Cache

  @doc "Start the in-memory response cache (idempotent)."
  def start_cache do
    case :ets.whereis(@cache) do
      :undefined -> :ets.new(@cache, [:named_table, :public, :set])
      _ -> @cache
    end

    :ok
  end

  @doc """
  List all tracks. Returns the raw track maps with `is_joined`, `num_exercises`,
  `num_completed_exercises`, `slug`, `title`, `web_url`, etc. Joined tracks first.
  """
  @spec tracks(Config.t(), keyword()) :: {:ok, [map()]} | {:error, term()}
  def tracks(config, opts \\ []) do
    with {:ok, body} <- get(config, "/tracks", opts) do
      tracks = Map.get(body, "tracks", [])
      {joined, rest} = Enum.split_with(tracks, &Map.get(&1, "is_joined", false))
      {:ok, joined ++ rest}
    end
  end

  @doc """
  List exercises for a track joined with the user's solutions. Each entry:

      %{
        slug: ..., title: ..., difficulty: ..., blurb: ...,
        is_unlocked: bool, is_recommended: bool,
        status: Xrc.Status.t(),
        solution: solution_map | nil,
        web_url: ...    # exercise page or solution page
      }
  """
  @spec exercises(Config.t(), String.t(), keyword()) :: {:ok, [map()]} | {:error, term()}
  def exercises(config, track, opts \\ []) do
    path = "/tracks/#{track}/exercises?sideload=solutions"

    with {:ok, body} <- get(config, path, opts) do
      exercises = Map.get(body, "exercises", [])
      solutions = Map.get(body, "solutions", [])

      by_slug =
        Map.new(solutions, fn sol ->
          {get_in(sol, ["exercise", "slug"]), sol}
        end)

      merged =
        Enum.map(exercises, fn ex ->
          slug = Map.get(ex, "slug")
          solution = Map.get(by_slug, slug)

          %{
            slug: slug,
            title: Map.get(ex, "title", slug),
            difficulty: Map.get(ex, "difficulty"),
            blurb: Map.get(ex, "blurb", ""),
            is_unlocked: Map.get(ex, "is_unlocked", true),
            is_recommended: Map.get(ex, "is_recommended", false),
            status: Status.derive(ex, solution),
            solution: solution,
            web_url: exercise_web_url(config, track, ex, solution)
          }
        end)

      {:ok, merged}
    end
  end

  @doc "Best web URL for an exercise: the user's solution page if any, else the exercise page."
  def exercise_web_url(_config, track, exercise, solution) do
    cond do
      solution && Map.get(solution, "public_url") -> Map.get(solution, "public_url")
      solution && Map.get(solution, "private_url") -> Map.get(solution, "private_url")
      url = get_in(exercise, ["links", "self"]) -> absolutize(url)
      true -> "https://exercism.org/tracks/#{track}/exercises/#{Map.get(exercise, "slug")}"
    end
  end

  defp absolutize("http" <> _ = url), do: url
  defp absolutize("/" <> _ = path), do: "https://exercism.org" <> path
  defp absolutize(other), do: other

  # --- HTTP ---

  defp get(config, path, opts) do
    refresh = Keyword.get(opts, :refresh, false)
    start_cache()
    key = {:get, path}

    case (not refresh && cache_get(key)) || nil do
      nil ->
        do_get(config, path, key)

      cached ->
        {:ok, cached}
    end
  end

  defp do_get(config, path, key) do
    url = Config.v2_base() <> path

    result =
      Req.get(url,
        auth: {:bearer, config.token},
        headers: [{"user-agent", "xrc (exercism personal tool)"}],
        retry: :transient
      )

    case result do
      {:ok, %{status: 200, body: body}} when is_map(body) ->
        cache_put(key, body)
        {:ok, body}

      {:ok, %{status: 200, body: body}} when is_binary(body) ->
        decoded = Jason.decode!(body)
        cache_put(key, decoded)
        {:ok, decoded}

      {:ok, %{status: status, body: body}} ->
        {:error, {:http, status, summarize(body)}}

      {:error, reason} ->
        {:error, {:request, reason}}
    end
  end

  defp summarize(body) when is_map(body), do: Map.get(body, "error", body)
  defp summarize(body), do: body

  defp cache_get(key) do
    case :ets.lookup(@cache, key) do
      [{^key, value}] -> value
      [] -> nil
    end
  end

  defp cache_put(key, value), do: :ets.insert(@cache, {key, value})
end
