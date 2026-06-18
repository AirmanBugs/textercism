#!/usr/bin/env bash
# Prepares a known state for the demo recording: downloads a demo exercise and
# writes a partially-correct solution so the test run shows a realistic mix of
# passing and failing tests. Safe to re-run.
set -euo pipefail

EXERCISE="${1:-bird-count}"

# Build the binary the demo will use.
( cd "$(dirname "$0")/.." && go build -o textercism . )

# Download the exercise (force is fine here — it's a throwaway demo exercise,
# NOT one you're solving for real).
textercism restart elixir "$EXERCISE" >/dev/null 2>&1 || \
  ( cd "$(dirname "$0")/.." && ./textercism restart elixir "$EXERCISE" >/dev/null 2>&1 )

# Write a partial solution: today/1 works, the rest are stubs -> mixed results.
WS="$(python3 -c "import json;print(json.load(open('$HOME/.config/exercism/user.json'))['workspace'])")"
cat > "$WS/elixir/$EXERCISE/lib/bird_count.ex" <<'EOF'
defmodule BirdCount do
  def today([]), do: nil
  def today([head | _]), do: head

  def increment_day_count(list) do
    # Please implement the increment_day_count/1 function
  end

  def has_day_without_birds?(list) do
    # Please implement the has_day_without_birds?/1 function
  end

  def total(list) do
    # Please implement the total/1 function
  end

  def busy_days(list) do
    # Please implement the busy_days/1 function
  end
end
EOF

echo "Demo state ready for elixir/$EXERCISE"
