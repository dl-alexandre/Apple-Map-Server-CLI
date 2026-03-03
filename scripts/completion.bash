#!/bin/bash
# ams shell completion script for bash
# Source this file in your .bashrc or .bash_profile

_ams_completion() {
    local cur prev words cword
    _init_completion || return

    # Get available commands
    local commands="help auth geocode reverse directions search cache snapshot ping version"
    local subcommands="token autocomplete stats clear"
    
    # Complete the first argument (command)
    if [ $cword -eq 1 ]; then
        COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
        return 0
    fi
    
    # Handle subcommands
    local command=${words[1]}
    
    case ${command} in
        auth)
            COMPREPLY=( $(compgen -W "token" -- ${cur}) )
            ;;
        geocode)
            COMPREPLY=( $(compgen -W "--json --limit --file --concurrency" -- ${cur}) )
            ;;
        reverse)
            COMPREPLY=( $(compgen -W "--limit --json" -- ${cur}) )
            ;;
        directions)
            COMPREPLY=( $(compgen -W "--mode --eta --json" -- ${cur}) )
            ;;
        search)
            if [ $cword -eq 2 ]; then
                COMPREPLY=( $(compgen -W "autocomplete --near --region --near-address --no-cache --limit --category --json" -- ${cur}) )
            fi
            ;;
        cache)
            COMPREPLY=( $(compgen -W "stats clear" -- ${cur}) )
            ;;
        snapshot)
            COMPREPLY=( $(compgen -W "--zoom --size --format --output" -- ${cur}) )
            ;;
        ping)
            COMPREPLY=( $(compgen -W "--request-id" -- ${cur}) )
            ;;
        *)
            ;;
    esac
}

complete -F _ams_completion ams
