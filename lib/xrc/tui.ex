defmodule Xrc.TUI do
  @moduledoc """
  Interactive terminal UI built on Owl: pick a track, browse exercises with
  status badges, then choose an action. Also provides the non-interactive
  `print_tracks/1` and `print_exercises/2` used by `xrc tracks` / `xrc list`.
  """

  alias Xrc.{Actions, Api, Config, Local, Status}

  # --- interactive entry points ---

  @doc "Full interactive flow: track picker -> exercise picker -> action menu."
  def run(%Config{} = config) do
    case pick_track(config) do
      nil -> :ok
      track -> run(config, track)
    end
  end

  @doc "Interactive exercise picker + action loop for a known track."
  def run(%Config{} = config, track) do
    case Api.exercises(config, track) do
      {:ok, exercises} -> exercise_loop(config, track, exercises)
      {:error, reason} -> fail("Could not fetch exercises: #{inspect(reason)}")
    end
  end

  defp pick_track(config) do
    case Api.tracks(config) do
      {:ok, tracks} ->
        joined = Enum.filter(tracks, &Map.get(&1, "is_joined", false))
        list = if joined == [], do: tracks, else: joined

        Owl.IO.puts("")

        Owl.IO.select(list,
          label: "Select a track",
          render_as: &render_track/1
        )
        |> Map.get("slug")

      {:error, reason} ->
        fail("Could not fetch tracks: #{inspect(reason)}")
        nil
    end
  end

  defp exercise_loop(config, track, exercises) do
    # Only unlocked exercises are actionable; show locked ones in the list view.
    actionable = Enum.reject(exercises, &(&1.status == :locked))

    Owl.IO.puts(["\n", Status.legend(), "\n"])

    selected =
      Owl.IO.select(actionable ++ [:back],
        label: "Select an exercise in #{track}",
        render_as: &render_exercise(&1, config, track)
      )

    case selected do
      :back -> :ok
      exercise -> action_menu(config, track, exercise)
    end
  end

  defp action_menu(config, track, exercise) do
    slug = exercise.slug
    downloaded = Local.downloaded?(config, track, slug)
    actions = actions_for(exercise.status, downloaded)

    Owl.IO.puts([
      "\n",
      Status.badge(exercise.status),
      " ",
      Owl.Data.tag(exercise.title, :bright_white),
      "  ",
      Owl.Data.tag("(#{Status.label(exercise.status)})", :light_black)
    ])

    if exercise.blurb != "", do: Owl.IO.puts([Owl.Data.tag(exercise.blurb, :light_black)])

    {_label, action} =
      Owl.IO.select(actions, label: "Action", render_as: fn {label, _} -> label end)

    perform(action, config, track, slug)
  end

  # Available actions depend on status + local state.
  defp actions_for(status, downloaded) do
    base =
      case {status, downloaded} do
        {:not_started, _} -> [{"Start (download + open VS Code)", :start}]
        {_, false} -> [{"Continue (download + open VS Code)", :start}]
        {_, true} -> [{"Continue (open VS Code)", :open}]
      end

    base ++
      [
        {"Run tests", :test},
        {"Submit", :submit},
        {"Re-download stubs (restart)", :restart},
        {"Open in browser", :web},
        {"Back", :back}
      ]
  end

  defp perform(:start, config, track, slug), do: Actions.start(config, track, slug, false)
  defp perform(:restart, config, track, slug), do: Actions.start(config, track, slug, true)
  defp perform(:open, config, track, slug), do: Actions.open(config, track, slug)
  defp perform(:test, config, track, slug), do: Actions.test(config, track, slug)
  defp perform(:web, config, track, slug), do: Actions.web(config, track, slug)
  defp perform(:back, _config, _track, _slug), do: :ok

  defp perform(:submit, config, track, slug) do
    Actions.submit(config, track, slug, fn q -> Owl.IO.confirm(message: q) end)
  end

  # --- non-interactive printing ---

  @doc "Print all tracks with join state and progress (`xrc tracks`)."
  def print_tracks(%Config{} = config) do
    case Api.tracks(config) do
      {:ok, tracks} ->
        Enum.each(tracks, fn t -> Owl.IO.puts(render_track(t)) end)

      {:error, reason} ->
        fail("Could not fetch tracks: #{inspect(reason)}")
    end
  end

  @doc "Print exercises for a track with status badges (`xrc list <track>`)."
  def print_exercises(%Config{} = config, track) do
    case Api.exercises(config, track) do
      {:ok, exercises} ->
        Owl.IO.puts([Status.legend(), "\n"])
        Enum.each(exercises, fn ex -> Owl.IO.puts(render_exercise(ex, config, track)) end)

      {:error, reason} ->
        fail("Could not fetch exercises: #{inspect(reason)}")
    end
  end

  # --- rendering ---

  defp render_track(track) do
    slug = Map.get(track, "slug")
    done = Map.get(track, "num_completed_exercises", 0)
    total = Map.get(track, "num_exercises", 0)
    joined = Map.get(track, "is_joined", false)

    marker = if joined, do: Owl.Data.tag("✔", :green), else: Owl.Data.tag("·", :light_black)
    progress = Owl.Data.tag("#{done}/#{total}", :light_black)

    [marker, " ", String.pad_trailing(slug, 22), " ", progress]
  end

  defp render_exercise(:back, _config, _track), do: Owl.Data.tag("← Back", :light_black)

  defp render_exercise(ex, config, track) do
    badge = Status.badge(ex.status)
    local = if Local.downloaded?(config, track, ex.slug), do: " ⬇", else: "  "
    rec = if ex.is_recommended, do: Owl.Data.tag(" ★rec", :magenta), else: ""

    diff =
      case ex.difficulty do
        nil -> ""
        d -> Owl.Data.tag(" [#{d}]", :light_black)
      end

    [badge, local, " ", String.pad_trailing(ex.title, 28), diff, rec]
  end

  defp fail(msg), do: Owl.IO.puts([Owl.Data.tag("✘ ", :red), msg])
end
