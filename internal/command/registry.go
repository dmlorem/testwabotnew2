package command

import (
	"fmt"
	"meowabot/internal/config"
	"meowabot/internal/database"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

var Default = &CommandList{Commands: make(map[string]*Command), Aliases: make([]string, 0)}

type CommandContext struct {
	Client    *whatsmeow.Client
	Config    *config.ConfigScheme
	Msg       *events.Message
	DB        *database.DBInstance
	Body      string
	Args      string
	Prefix    string
	Command   string
	Localizer *i18n.Localizer
	Log       *zerolog.Logger
}

type Requirements struct {
	BotAdmin bool
	Mention  bool
}

type Only struct {
	Owner   bool
	Admin   bool
	Group   bool
	Premium bool
}

type Command struct {
	Aliases []string
	Run     func(ctx *CommandContext) error
	Need    Requirements
	Only    Only
}

type CommandList struct {
	Commands map[string]*Command
	Aliases  []string
}

func (r *CommandList) Register(cmd *Command) {
	for _, a := range cmd.Aliases {
		if _, ok := r.Commands[a]; ok {
			panic(fmt.Sprintf("Duplicate command %s", a))
		}
		r.Commands[a] = cmd
	}
	r.Aliases = append(r.Aliases, cmd.Aliases...)
}
