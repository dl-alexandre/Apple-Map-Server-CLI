#!/bin/zsh
# ams shell completion script for zsh
# Place this file in your fpath (e.g., /usr/local/share/zsh/site-functions/_ams)

_ams() {
    local curcontext="$curcontext" state line
    typeset -A opt_args

    _arguments -C \
        '(-): :->command' \
        '(-)*:: :->args' && return 0

    case "$state" in
        command)
            local commands=(
                'help:Show help for a command'
                'auth:Exchange JWT for access token'
                'geocode:Geocode an address'
                'reverse:Reverse geocode coordinates'
                'directions:Get directions between locations'
                'search:Search for places and POIs'
                'cache:Manage geocode cache'
                'snapshot:Generate static map image'
                'ping:Ping Apple Map Server'
                'version:Show version info'
            )
            _describe -t commands 'ams command' commands
            ;;
        args)
            case "$line[1]" in
                auth)
                    local auth_subcommands=(
                        'token:Exchange JWT for access token'
                    )
                    _describe -t commands 'auth subcommand' auth_subcommands
                    ;;
                geocode)
                    _arguments \
                        '--json[Output raw JSON response]' \
                        '--limit[Maximum number of results]' \
                        '--file[Path to file with one query per line]' \
                        '--concurrency[Number of concurrent requests]'
                    ;;
                reverse)
                    _arguments \
                        '--limit[Maximum number of results]' \
                        '--json[Output raw JSON response]'
                    ;;
                directions)
                    _arguments \
                        '--mode[Transport mode: car, walk, transit, bike]' \
                        '--eta[Show only ETA and distance]' \
                        '--json[Output raw JSON response]'
                    ;;
                search)
                    _arguments \
                        '(-): :->search_subcommand' && return 0
                    
                    case "$line[2]" in
                        autocomplete)
                            _arguments \
                                '--near[Center point for location bias]' \
                                '--limit[Maximum number of suggestions]' \
                                '--json[Output raw JSON response]'
                            ;;
                        *)
                            _arguments \
                                '--near[Center point for search as lat,lng]' \
                                '--region[Bounding box as n,e,s,w]' \
                                '--near-address[Address to geocode and search around]' \
                                '--no-cache[Bypass geocode cache]' \
                                '--limit[Maximum number of results]' \
                                '--category[Filter by POI category]' \
                                '--json[Output raw JSON response]'
                            ;;
                    esac
                    ;;
                cache)
                    local cache_subcommands=(
                        'stats:Show cache statistics'
                        'clear:Clear all cached entries'
                    )
                    _describe -t commands 'cache subcommand' cache_subcommands
                    ;;
                snapshot)
                    _arguments \
                        '--zoom[Zoom level 1-20]:zoom level:(1 5 10 12 15 20)' \
                        '--size[Image dimensions WxH]:size:(300x200 600x400 800x600 1200x800)' \
                        '--format[Output format]:format:(png jpg)' \
                        '--output[Output file path]:file:_files'
                    ;;
                ping)
                    _arguments \
                        '--request-id[Include request ID in output]'
                    ;;
                *)
                    ;;
            esac
            ;;
    esac
}

_ams "$@"
