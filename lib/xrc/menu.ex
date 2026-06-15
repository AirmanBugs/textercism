defmodule Xrc.Menu do
  @moduledoc """
  Arrow-key driven single-select menu. Owl's `select` is numeric-entry only, and
  the BEAM's managed stdio server swallows raw-mode reads inside an escript, so
  this reads keystrokes by shelling out to the controlling terminal (`/dev/tty`)
  with `stty`/`dd`. Each keypress runs one short `sh` command attached to the tty.

  Navigation: ↑/↓ (or k/j) to move, Enter to confirm, q/Esc/Ctrl-C to cancel.

  Falls back to numbered input when there's no interactive tty (e.g. piped
  stdin), so non-interactive use and tests still work.
  """

  @doc """
  Prompt the user to pick one item from `items`.

  Options:
    * `:label`  — heading printed above the list
    * `:render` — 1-arg fun returning Owl data for an item (default: inspect)

  Returns the chosen item, or `nil` if cancelled.
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

  # --- interactive ---

  defp interactive_select(items, label, render) do
    hide_cursor()

    try do
      loop(items, label, render, 0, false)
    after
      show_cursor()
    end
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
      pointer = if selected?, do: Owl.Data.tag("❯ ", :cyan), else: "  "
      line = render.(item)
      line = if selected?, do: Owl.Data.tag(line, :bright), else: line
      Owl.IO.puts([pointer, line])
    end)
  end

  # Total terminal rows the menu occupies (label line + one per item).
  defp rows(items, label), do: length(items) + if(label, do: 1, else: 0)

  defp move_cursor_up(0), do: :ok
  defp move_cursor_up(n), do: IO.write("\e[#{n}A\e[0J")

  defp inc(i, len), do: rem(i + 1, len)
  defp dec(i, len), do: rem(i - 1 + len, len)

  # --- key reading via /dev/tty ---

  # Read one logical keypress by reading raw bytes from the controlling terminal.
  # `head -c` would block on partial escape sequences, so we read up to 3 bytes
  # with a short inter-byte timeout (`stty time 1`, tenths of a second) and let
  # the shell return whatever arrived. Returns :up|:down|:enter|:cancel|:other.
  defp read_key do
    script = ~S"""
    old=$(stty -g < /dev/tty)
    stty raw -echo min 1 time 1 < /dev/tty
    dd if=/dev/tty bs=3 count=1 2>/dev/null | od -An -tu1
    stty "$old" < /dev/tty
    """

    case sh(script) do
      {out, 0} -> classify(parse_bytes(out))
      _ -> :cancel
    end
  end

  defp parse_bytes(out) do
    out
    |> String.split(~r/\s+/, trim: true)
    |> Enum.map(&String.to_integer/1)
  end

  # Carriage return / newline -> enter; ESC [ A/B -> up/down; lone ESC, q, Ctrl-C
  # -> cancel; vim keys k/j -> up/down.
  defp classify([13 | _]), do: :enter
  defp classify([10 | _]), do: :enter
  defp classify([27, 91, 65 | _]), do: :up
  defp classify([27, 91, 66 | _]), do: :down
  defp classify([27]), do: :cancel
  defp classify([3 | _]), do: :cancel
  defp classify([?q | _]), do: :cancel
  defp classify([?Q | _]), do: :cancel
  defp classify([?k | _]), do: :up
  defp classify([?j | _]), do: :down
  defp classify([]), do: :other
  defp classify(_), do: :other

  # --- tty helpers ---

  defp interactive_tty? do
    match?({_, 0}, sh("stty -g < /dev/tty"))
  rescue
    _ -> false
  end

  defp sh(command), do: System.cmd("sh", ["-c", command], stderr_to_stdout: true)

  defp hide_cursor, do: IO.write("\e[?25l")
  defp show_cursor, do: IO.write("\e[?25h")

  # --- non-interactive fallback ---

  defp numbered_select(items, label, render) do
    Owl.IO.select(items, label: label, render_as: render)
  end
end
