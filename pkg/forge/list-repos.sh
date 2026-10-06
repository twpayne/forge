#!/bin/sh

set -e

maxdepth=12

is_command() {
	type "${1}" >/dev/null 2>&1
}

if is_command fdfind; then
    fdfind --absolute-path --case-sensitive --color=never --max-depth="${maxdepth}" --print0 --unrestricted "^\.git$" "${HOME}/projects" "${HOME}/src"
elif is_command fd; then
    fd --absolute-path --case-sensitive --color=never --max-depth="${maxdepth}" --print0 --unrestricted "^\.git$" "${HOME}/projects" "${HOME}/src"
elif is_command find; then
    find "${HOME}/projects" "${HOME}/src" -maxdepth "${maxdepth}" -name .git -print0
fi