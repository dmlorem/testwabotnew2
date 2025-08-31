package commands

import (
	"context"
	"meowabot/internal/command"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func init() {
	cmd := command.Default
	cmd.Register(&command.Command{
		Aliases: []string{"ping"},
		Run: func(ctx *command.CommandContext) error {
			msg, err := ctx.Client.SendMessage(context.Background(), ctx.Msg.Info.Chat, &waE2E.Message{
				ExtendedTextMessage: &waE2E.ExtendedTextMessage{
					Text: proto.String(ctx.Localizer.MustLocalize(&i18n.LocalizeConfig{
						DefaultMessage: &i18n.Message{
							ID:    "cmd.ping",
							Other: "Pong!",
						},
					})),
					ContextInfo: &waE2E.ContextInfo{
						StanzaID:      &ctx.Msg.Info.ID,
						Participant:   proto.String(ctx.Msg.Info.Sender.String()),
						QuotedMessage: ctx.Msg.Message,
					},
				},
			})
			if err != nil {
				return err
			}

			t := msg.Timestamp.Sub(ctx.Msg.Info.Timestamp)

			ctx.SendTextMessage(ctx.Msg.Info.Chat, ctx.Localizer.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "cmd.ping-speed",
					Other: "Velocidade de resposta: {{.Speed}}",
				},
				TemplateData: map[string]any{
					"Speed": t.Milliseconds(),
				},
			}), &command.MessageOptions{
				QuotedMessage: ctx.Msg,
			})
			return nil
		},
	})
}
