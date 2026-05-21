package cmd

import (
	"fmt"
)

type Command struct {
	Name    string
	MinArgs int
	Desc    string
	Do      func(args []string) error
}

func (c *Command) Usage() string { return fmt.Sprintf("%s %s", c.Name, c.Desc) }
func Register(commands []Command) map[string]Command {
	m := make(map[string]Command, len(commands))
	for _, c := range commands {
		m[c.Name] = c
	}

	return m
}
