BIN_NAME := words-rain
CONFIG_DIR := $(HOME)/.config/words-rain
CONFIG_FILE := $(CONFIG_DIR)/config.env

.DEFAULT_GOAL := all

.PHONY: all build install clean

all: clean build install

build:
	go build -o $(BIN_NAME) .

install: build
	mkdir -p "$(HOME)/bin"
	cp -f "$(BIN_NAME)" "$(HOME)/bin/$(BIN_NAME)"
	mkdir -p "$(CONFIG_DIR)"
	mkdir -p "$(CONFIG_DIR)/wordbooks"
	if ls wordbooks/*.txt >/dev/null 2>&1; then cp -f wordbooks/*.txt "$(CONFIG_DIR)/wordbooks/"; fi
	printf '%s\n' \
		'# words-rain default config' \
		'WORDS_RAIN_HOST=127.0.0.1' \
		'WORDS_RAIN_PORT=8080' \
		"WORDS_RAIN_WORDBOOKS_DIR=$(CONFIG_DIR)/wordbooks" \
		> "$(CONFIG_FILE)"

clean:
	rm -f "$(BIN_NAME)"
