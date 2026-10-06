#!/bin/fish

set -l max_depth 12

function is_command
    type -q $argv[1]
end

if is_command fdfind
    fdfind --absolute-path --case-sensitive --color=never --max-depth=$max_depth --print0 --unrestricted "^\.git\$" "$HOME/projects" "$HOME/src"
else if is_command fd
    fd --absolute-path --case-sensitive --color=never --max-depth=$max_depth --print0 --unrestricted "^\.git\$" "$HOME/projects" "$HOME/src"
else if is_command find
    find "$HOME/projects" "$HOME/src" -maxdepth $max_depth -name .git -print0
end