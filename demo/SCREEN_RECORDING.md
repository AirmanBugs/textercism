# Screen-recording demo (split screen: editor/browser left, terminal right)

This shows the full integration — textercism opening VS Code and the browser —
which a terminal-only recorder (vhs) can't capture. You drive it live and record
your real screen.

## One-time setup

```sh
brew install gifski ffmpeg     # GIF conversion (gifski = best quality)
```

## 1. Arrange the windows (split screen)

- **Right half:** iTerm, running `textercism`. Make the font large (the GIF will be
  scaled down — big text stays legible). iTerm > hold ⌥ and drag, or use a window
  manager, to snap it to the right half.
- **Left half:** VS Code on top, your browser below it (or side by side in the left
  half). These start empty; textercism will open the exercise into them during the demo.

Tip: close notifications, hide the menu bar clutter, and pick a clean terminal theme.

## 2. Prepare the exercise state

```sh
cd ~/Source/textercism
./demo/setup.sh bird-count      # partial solution: today/1 solved, rest stubbed
```

This gives a realistic mixed test result. bird-count is a **throwaway demo exercise**
here (setup force-overwrites it) — don't use one you're solving for real.

> For the **submit** beat: submit only goes through when all tests pass. Either
> (a) fully solve bird-count first so submit succeeds, or (b) show that submit runs
> the tests and reports "tests failed — not submitted" (also a fair thing to show).

## 3. Record (built-in macOS capture)

1. Press **⌘⇧5**.
2. Choose **Record Selected Portion**, and drag the selection to cover **both halves**
   (the whole split layout).
3. Click **Record**. Run through the shot list below.
4. Click the **stop** button in the menu bar. The `.mov` saves to your Desktop.

Keep it to **~20–35 seconds** — long enough to show the flow, short enough for Slack.

## 4. Shot list (what to do on camera)

1. **Launch** — `textercism elixir` in iTerm. (2s)
2. **Browse** — arrow down a few exercises to show the status badges. (3s)
3. **Open bird-count** — filter/scroll to Bird Count, Enter. (2s)
4. **Instructions** — it's highlighted by default; Tab into the pane, scroll. (3s)
5. **Hints** — arrow to Hints, press `n` two or three times to reveal hints, then `o`
   to **open a doc link** — the browser on the left jumps to the Elixir docs. (5s)
6. **Run tests** — arrow to Run tests, Enter. Show the clean pass/fail list; press `a`
   to expand an assertion. (4s)
7. **Open in VS Code** — arrow to Continue, Enter. **VS Code on the left opens** the
   solution. (3s)
8. **Open in browser** — arrow to Open in browser, Enter. The browser shows the
   exercism.org page. (2s)
9. *(optional)* **Restart** — arrow to Restart, Enter, and show VS Code reload the
   fresh stub. (3s)
10. *(optional)* **Submit** — arrow to Submit, Enter. Either it submits (if solved) or
    reports tests must pass. (3s)

## 5. Convert .mov → optimized GIF for Slack

Replace `INPUT.mov` with your file. This scales to 1280px wide, ~12 fps, and uses
gifski for clean dithering:

```sh
# extract frames at 12fps, scaled to 1280px wide, then gifski -> gif
ffmpeg -i INPUT.mov -vf "fps=12,scale=1280:-1:flags=lanczos" demo/frames/%04d.png
gifski --fps 12 --width 1280 --quality 80 -o demo/textercism.gif demo/frames/*.png
rm -rf demo/frames
```

Quick one-liner alternative (ffmpeg-only, larger/lower quality):

```sh
ffmpeg -i INPUT.mov -vf "fps=12,scale=1280:-1:flags=lanczos" demo/textercism.gif
```

Aim for **under ~3 MB** so Slack plays it inline. If it's too big: drop `--width` to
1000, `--fps` to 10, or trim the clip.

## 6. Post to Slack

Drag `demo/textercism.gif` into the message (or paste from clipboard). GIFs
**auto-play inline** for everyone — no click needed.
