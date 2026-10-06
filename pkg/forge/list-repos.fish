#!/bin/fish

function is_command
    type -q $argv[1]
end

if is_command fdfind
    fdfind --absolute-path --case-sensitive --color=never --max-depth=4 --print0 --unrestricted "^\.git\$" "$HOME/projects" "$HOME/src"
else if is_command fd
    fd --absolute-path --case-sensitive --color=never --max-depth=4 --print0 --unrestricted "^\.git\$" "$HOME/projects" "$HOME/src"
else if is_command find
    find "$HOME/projects" "$HOME/src" -maxdepth 4 -name .git -print0
end