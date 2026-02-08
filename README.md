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
./words-rain
```

If a required setting is still missing (for example wordbooks directory), startup fails with a clear error message.

## Build

```bash
go build -o typing-rain .
./typing-rain --wordbooks-dir ./wordbooks --port 8080
```

Or use `make`:

```bash
make build
```

## Install

```bash
make install
```

This will:

- Build `words-rain`.
- Copy it to `~/bin/words-rain`.
- Create `~/.config/words-rain`.
- Create default config at `~/.config/words-rain/config.env` if it does not exist.

## Cross Compile

macOS (Apple Silicon):

```bash
GOOS=darwin GOARCH=arm64 go build -o typing-rain-macos-arm64 .
```

Windows x64:

```bash
GOOS=windows GOARCH=amd64 go build -o typing-rain-windows-amd64.exe .
```

Linux x64:

```bash
GOOS=linux GOARCH=amd64 go build -o typing-rain-linux-amd64 .
```

Then run with:

```bash
./typing-rain-<target> --wordbooks-dir <path-to-wordbooks>
```
