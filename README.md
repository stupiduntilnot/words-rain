# Typing Rain Game

A framework-free browser typing game served by a single Go binary.

## Features

- Real-time letter matching against falling words.
- Target selection rule: nearest-to-ground word wins when multiple prefixes match.
- Combo scoring: `+1, +2, +3...`; combo resets when a word reaches the ground.
- Word speech after each successful elimination with game-wide bullet-time pause.
- Dissolve animation when a word is completed.
- Wordbooks loaded from a folder of `.txt` files (filename is the selectable name).
- Game ends when all words are eventually cleared and no missed words remain.

## Wordbook Format

Use one `.txt` file per wordbook.

- One word per line.
- Empty lines are ignored.
- Words are normalized to lowercase for comparison and rendering.

Example: `wordbooks/letters.txt`.

## Run

```bash
go run . --wordbooks-dir ./wordbooks --port 8080
```

Open `http://127.0.0.1:8080`.

If you run with no flags, the app loads defaults from:

`~/.config/words-rain/config.env`

```bash
words-rain
```

If a required setting is still missing (for example wordbooks directory), startup fails with a clear error message.

`WORDS_RAIN_OPEN_BROWSER=true` in config enables auto-opening the browser on startup.
`WORDS_RAIN_ACCENT=en-US` sets the default TTS accent in the setup UI.

CLI flags:

- `--wordbooks-dir` (required unless loaded from default config in no-flag mode)
- `--host` (default `127.0.0.1`)
- `--port` (default `8080`)
- `--open-browser`

Important behavior:

- Default config loading happens only when no CLI flags are provided.
- The setup page accent selection is persisted to config via backend API and restored on next launch.

## Build

```bash
go build -o words-rain .
./words-rain --wordbooks-dir ./wordbooks --port 8080
```

Make targets:

```bash
make          # clean + build + install
make build
make install
make clean
```

## Install

```bash
make install
```

This will:

- Build `words-rain`.
- Copy it to `~/bin/words-rain`.
- Ensure user execute permission on `~/bin/words-rain`.
- Create `~/.config/words-rain`.
- Copy all `wordbooks/*.txt` into `~/.config/words-rain/wordbooks`.
- Overwrite default config at `~/.config/words-rain/config.env`.

Default config content:

```env
# words-rain default config
WORDS_RAIN_HOST=127.0.0.1
WORDS_RAIN_PORT=8080
WORDS_RAIN_OPEN_BROWSER=true
WORDS_RAIN_ACCENT=en-US
WORDS_RAIN_WORDBOOKS_DIR=~/.config/words-rain/wordbooks
```

## Cross Compile

macOS (Apple Silicon):

```bash
GOOS=darwin GOARCH=arm64 go build -o words-rain-macos-arm64 .
```

Windows x64:

```bash
GOOS=windows GOARCH=amd64 go build -o words-rain-windows-amd64.exe .
```

Linux x64:

```bash
GOOS=linux GOARCH=amd64 go build -o words-rain-linux-amd64 .
```

Then run with:

```bash
./words-rain-<target> --wordbooks-dir <path-to-wordbooks>
```
