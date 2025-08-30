package commands

import (
	"meowabot/internal/command"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func init() {
	cmd := command.Default
	cmd.Register(&command.Command{
		Aliases: []string{"ping"},
		Run: func(ctx *command.CommandContext) error {
			ctx.SendTextMessage(ctx.Msg.Info.Chat, ctx.Localizer.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "pingcmd",
					Other: "Pong!",
				},
			}), &command.MessageOptions{
				QuotedMessage: ctx.Msg,
			})
			return nil
		},
	})
}
