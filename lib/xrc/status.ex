defmodule Xrc.Status do
  @moduledoc """
  Derives a single exercise status from the unofficial v2 API shape: an
  `exercise` map (with `is_unlocked`) and an optional `solution` map (with a
  `status` string). The schema is unofficial, so parsing is deliberately lenient.
  """

  @type t :: :not_started | :in_progress | :completed | :published | :locked

  @doc """
  Map an exercise + optional solution to a status atom.

    * no solution and locked -> :locked
    * no solution and unlocked -> :not_started
    * solution "started"/"iterated" -> :in_progress
    * solution "completed" -> :completed
    * solution "published" -> :published
    * any other non-nil solution status -> :in_progress (lenient fallback)
  """
  @spec derive(map(), map() | nil) :: t()
  def derive(exercise, solution)

  def derive(exercise, nil) do
    if Map.get(exercise, "is_unlocked", true), do: :not_started, else: :locked
  end

  def derive(_exercise, solution) when is_map(solution) do
    case Map.get(solution, "status") do
      "completed" -> :completed
      "published" -> :published
      "started" -> :in_progress
      "iterated" -> :in_progress
      _ -> :in_progress
    end
  end

  @badges %{
    not_started: {"●", :light_black, "not started"},
    in_progress: {"◐", :yellow, "in progress"},
    completed: {"✓", :green, "completed"},
    published: {"★", :cyan, "published"},
    locked: {"🔒", :red, "locked"}
  }

  @doc "An Owl-colored badge tag for a status."
  def badge(status) do
    {glyph, color, _label} = Map.fetch!(@badges, status)
    Owl.Data.tag(glyph, color)
  end

  @doc "Human label for a status."
  def label(status) do
    {_glyph, _color, label} = Map.fetch!(@badges, status)
    label
  end

  @doc "Legend line for the exercise list footer."
  def legend do
    [:not_started, :in_progress, :completed, :published, :locked]
    |> Enum.map(fn s -> [badge(s), " ", label(s)] end)
    |> Enum.intersperse("   ")
  end
end
