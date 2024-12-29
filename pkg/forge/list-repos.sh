#!/bin/sh

set -e

is_command() {
	type "${1}" >/dev/null 2>&1
}

if is_command fdfind; then
    fdfind --absolute-path --case-sensitive --color=never --max-depth=4 --print0 --type=directory --unrestricted "^\.git$" "${HOME}/src"
elif is_command fd; then
    fd --absolute-path --case-sensitive --color=never --max-depth=4 --print0 --type=directory --unrestricted "^\.git$" "${HOME}/src"
elif is_command find; then
    find "${HOME}/src" -maxdepth 4 -name .git -type d -print0
fi