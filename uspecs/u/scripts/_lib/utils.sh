#!/usr/bin/env bash

# checkcmds command1 [command2 ...]
# Verifies that each listed command is available on PATH.
# Prints an error message and exits with status 1 if any command is missing.
checkcmds() {
    local cmd
    for cmd in "$@"; do
        if ! command -v "$cmd" > /dev/null 2>&1; then
            echo "Error: required command not found: $cmd" >&2
            exit 1
        fi
    done
}
