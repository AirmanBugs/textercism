# textercism demo — recorded with vhs (https://github.com/charmbracelet/vhs)
#
#   1. brew install vhs
#   2. ./demo/setup.sh          # prepares a partially-solved demo exercise
#   3. vhs demo/demo.tape       # renders demo/textercism.gif
#
# Each "Sleep" gives viewers time to read. Trim/retime sections freely — the
# blocks below are labelled so you can cut the ones you don't want.

Output demo/textercism.gif

# --- terminal look ---
Set Shell "bash"
Set FontSize 18
Set Width 1400
Set Height 800
Set Padding 20
Set Theme "Catppuccin Mocha"
Set PlaybackSpeed 1.0

# --- launch into the elixir track ---
Type "textercism elixir"
Enter
Sleep 2s

# === BLOCK 1: browse exercises (status badges) ===
# Scroll through the exercise list to show the status indicators.
Down
Sleep 600ms
Down
Sleep 600ms
Down
Sleep 800ms
Up@600ms 3
Sleep 500ms

# Open the demo exercise (bird-count is the in-progress one).
# Adjust the number of Downs if it's not highlighted yet.
Type "/bird"
Sleep 800ms
Enter
Sleep 600ms
Enter
Sleep 2s

# === BLOCK 2: instructions pane ===
# Instructions is highlighted by default — show the rendered README.
Tab
Sleep 1s
Down@250ms 6
Sleep 1.5s
Tab
Sleep 500ms

# === BLOCK 3: hints, revealed one at a time ===
Down
Sleep 1.5s
Type "n"
Sleep 1s
Type "n"
Sleep 1s
Type "n"
Sleep 1.5s

# === BLOCK 4: run tests, see the clean results ===
Down
Sleep 1s
Enter
Sleep 2.5s
# Expand assertion detail on the failures.
Type "a"
Sleep 1s
Tab
Down@250ms 8
Sleep 2s
Tab
Sleep 500ms

# === BLOCK 5 (optional — cut if too long): open in editor / browser ===
# Down to "Continue" and open VS Code, or skip.
# Down
# Enter
# Sleep 2s

# --- exit ---
Escape
Sleep 500ms
Escape
Sleep 500ms
Type "q"
Sleep 1s
