defmodule Xrc.Menu do
  @moduledoc """
  Arrow-key driven single-select menu. Owl's `select` is numeric-entry only, so
  this reads raw keystrokes (via `stty` raw mode) and renders a highlighted list
  that the user moves through with ↑/↓ (or k/j) and confirms with Enter.

  Falls back to numbered input when stdin isn't an interactive TTY (e.g. piped),
  so non-interactive use and tests still work.
  """

  @doc """
  Prompt the user to pick one item from `items`.

  Options:
    * `:label`     — heading printed above the list
    * `:render`    — 1-arg fun returning Owl data for an item (default: inspect)

  Returns the chosen item, or `nil` if cancelled (q / Esc / Ctrl-C).
  """
  def select(items, opts \\ [])

  def select([], _opts), do: nil

  def select(items, opts) do
    label = Keyword.get(opts, :label)
    render = Keyword.get(opts, :render, &inspect/1)

    if interactive_tty?() do
      interactive_select(items, label, render)
    else
      numbered_select(items, label, render)
    end
  end

  # --- interactive (raw mode) ---

  defp interactive_select(items, label, render) do
    with_raw_mode(fn ->
      loop(items, label, render, 0, false)
    end)
  end

  defp loop(items, label, render, cursor, drawn?) do
    if drawn?, do: move_cursor_up(rows(items, label))
    draw(items, label, render, cursor)

    case read_key() do
      :up -> loop(items, label, render, dec(cursor, length(items)), true)
      :down -> loop(items, label, render, inc(cursor, length(items)), true)
      :enter -> Enum.at(items, cursor)
      :cancel -> nil
      _other -> loop(items, label, render, cursor, true)
    end
  end

  defp draw(items, label, render, cursor) do
    if label, do: Owl.IO.puts(Owl.Data.tag(label, :cyan))

    items
    |> Enum.with_index()
    |> Enum.each(fn {item, idx} ->
      selected? = idx == cursor

      pointer =
        if selected?, do: Owl.Data.tag("❯ ", :cyan), else: "  "

      line = render.(item)
      line = if selected?, do: Owl.Data.tag(line, :bright), else: line
      Owl.IO.puts([pointer, line])
    end)
  end

  # Total terminal rows the menu occupies (label + one per item).
  defp rows(items, label) do
    length(items) + if(label, do: 1, else: 0)
  end

  defp move_cursor_up(0), do: :ok
  defp move_cursor_up(n), do: IO.write("\e[#{n}A\e[0J")

  defp inc(i, len), do: rem(i + 1, len)
  defp dec(i, len), do: rem(i - 1 + len, len)

  # --- key reading ---

  # Returns :up | :down | :enter | :cancel | :other
  defp read_key do
    case IO.binread(:stdio, 1) do
      "\r" -> :enter
      "\n" -> :enter
      "\e" -> read_escape()
      # Ctrl-C, q, Q, Esc handled as cancel
      <<3>> -> :cancel
      "q" -> :cancel
      "Q" -> :cancel
      "k" -> :up
      "j" -> :down
      :eof -> :cancel
      _ -> :other
    end
  end

  defp read_escape do
    case IO.binread(:stdio, 1) do
      "[" ->
        case IO.binread(:stdio, 1) do
          "A" -> :up
          "B" -> :down
          _ -> :other
        end

      # lone Esc
      _ ->
        :cancel
    end
  end

  # --- raw mode handling ---

  defp with_raw_mode(fun) do
    saved = stty(["-g"]) |> String.trim()
    stty(["raw", "-echo"])
    hide_cursor()

    try do
      fun.()
    after
      show_cursor()
      if saved != "", do: stty([saved]), else: stty(["sane"])
      IO.write("\n")
    end
  end

  defp interactive_tty? do
    # `stty` against the controlling terminal succeeds only when one exists.
    match?({_, 0}, sh("stty -g < /dev/tty"))
  rescue
    _ -> false
  end

  # Run stty against /dev/tty so it controls the real terminal regardless of how
  # the BEAM wired the escript's stdin.
  defp stty(args) do
    case sh("stty #{Enum.join(args, " ")} < /dev/tty") do
      {out, _} -> out
    end
  rescue
    _ -> ""
  end

  defp sh(command), do: System.cmd("sh", ["-c", command], stderr_to_stdout: true)

  defp hide_cursor, do: IO.write("\e[?25l")
  defp show_cursor, do: IO.write("\e[?25h")

  # --- non-interactive fallback ---

  defp numbered_select(items, label, render) do
    Owl.IO.select(items, label: label, render_as: render)
  end
end
