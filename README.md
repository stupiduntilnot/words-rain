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

## Build

```bash
go build -o typing-rain .
./typing-rain --wordbooks-dir ./wordbooks --port 8080
```

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
