package main

import (
	"html/template"
	"os"

	"github.com/urfave/cli/v2"
)

const zsh_completion_script = `
#compdef {{.}} 

_cli_zsh_autocomplete() {
  local -a opts
  local cur
  cur=${words[-1]}
  if [[ "$cur" == "-"* ]]; then
    opts=("${(@f)$(${words[@]:0:#words[@]-1} ${cur} --generate-bash-completion)}")
  else
    opts=("${(@f)$(${words[@]:0:#words[@]-1} --generate-bash-completion)}")
  fi

  if [[ "${opts[1]}" != "" ]]; then
    _describe 'values' opts
  else
    _files
  fi
}

compdef _cli_zsh_autocomplete {{.}}
`

const bash_completion_script = `
#! /bin/bash

# Macs have bash3 for which the bash-completion package doesn't include
# _init_completion. This is a minimal version of that function.
_cli_init_completion() {
  COMPREPLY=()
  _get_comp_words_by_ref "$@" cur prev words cword
}

_cli_bash_autocomplete() {
  if [[ "${COMP_WORDS[0]}" != "source" ]]; then
    local cur opts base words
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    if declare -F _init_completion >/dev/null 2>&1; then
      _init_completion -n "=:" || return
    else
      _cli_init_completion -n "=:" || return
    fi
    words=("${words[@]:0:$cword}")
    if [[ "$cur" == "-"* ]]; then
      requestComp="${words[*]} ${cur} --generate-bash-completion"
    else
      requestComp="${words[*]} --generate-bash-completion"
    fi
    opts=$(eval "${requestComp}" 2>/dev/null)
    COMPREPLY=($(compgen -W "${opts}" -- ${cur}))
    return 0
  fi
}

complete -o bashdefault -o default -o nospace -F _cli_bash_autocomplete {{.}}
`

func completionCommand() *cli.Command {
	return &cli.Command{
		Name:    "completion",
		Aliases: []string{"comp"},
		Usage:   "Generate shell completion scripts",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "shell",
			},
		},
		Action: func(c *cli.Context) error {
			shell := c.String("shell")
			var script string
			var prog string
			switch shell {
			case "bash":
				script = bash_completion_script
				prog = os.Args[0]
			case "zsh":
				script = zsh_completion_script
				prog = c.App.Name
			}
			return template.Must(template.New("completion_script").Parse(script)).Execute(os.Stdout, prog)
		},
	}
}
