# bash completion for recopy
_recopy_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="doctor --move --dry-run --mirror --profile --no-reflink --inplace --parallel --transport --prescan --verify --one-file-system --no-ui --help --version"
    if [[ ${cur} == -* ]]; then
        COMPREPLY=( $(compgen -W "${opts}" -- "${cur}") )
        return 0
    fi
    case "${prev}" in
        --profile)
            COMPREPLY=( $(compgen -W "auto lan wan" -- "${cur}") )
            return 0
            ;;
        --transport)
            COMPREPLY=( $(compgen -W "auto rsync btrfs" -- "${cur}") )
            return 0
            ;;
    esac
    return 0
}
complete -F _recopy_completions recopy
