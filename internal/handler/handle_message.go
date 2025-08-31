package handler

import (
	"context"
	"fmt"
	"meowabot/internal/command"
	"meowabot/internal/database"
	tmsg "meowabot/internal/tools/messages"
	"meowabot/internal/util"
	"slices"
	"strings"
	"time"

	"github.com/hbakhtiyor/strsim"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (i *EventHandler) handleMessage(m *events.Message) {
	if m.Info.IsFromMe || (m.Info.Chat.Server != types.DefaultUserServer && m.Info.Chat.Server != types.GroupServer && m.Info.Chat.Server != types.LegacyUserServer) {
		return
	}
	messageBody, isValid := tmsg.GetMessageText(m.Message)
	var prefix string = i.Config.CommandPrefix
	var isCommand bool = strings.HasPrefix(messageBody, i.Config.CommandPrefix)
	var commandName string
	var commandArgs string
	fmt.Println(commandArgs)
	if isCommand {
		commandName = util.NormalizeString(strings.ToLower(strings.Split(strings.TrimSpace(strings.TrimPrefix(messageBody, prefix)), " ")[0]))
		commandArgs = strings.TrimSpace(messageBody[len(commandName)+len(prefix):])
	}
	var isOwner bool = slices.Contains(i.Config.OwnerNumbers, m.Info.Sender.User)
	var isGroupAdmin bool
	var isBotGroupAdmin bool

	var groupMetadata *types.GroupInfo
	var userInfo *database.User
	var groupInfo *database.Group
	var participant *database.GroupParticipant

	err := func() error {
		i.UserDB.MU.Lock()
		defer i.UserDB.MU.Unlock()

		// User
		var err error
		userInfo, err = i.UserDB.GetUserInfo(m.Info.Sender.User)
		if err != nil {
			i.Log.Error().Err(err).Str("User", m.Info.Sender.User).Msg("Error retrieving user from database")
			return err
		}
		if userInfo.Name != m.Info.PushName {
			userInfo.Name = m.Info.PushName
		}
		if isCommand && !m.Info.IsGroup {
			userInfo.CommandCount++
		}

		err = i.UserDB.SaveUserInfo(userInfo)
		if err != nil {
			i.Log.Error().Err(err).Str("User", m.Info.Sender.User).Msg("Error saving user info")
			return err
		}

		if !i.receivedOldEvents.Load() {
			return fmt.Errorf("old message")
		}

		// Group info, group member info and anti spams
		if m.Info.IsGroup {
			// group metadata
			if cachedGroupInfo, ok := i.GetCachedGroupInfo(m.Info.Chat); ok {
				groupMetadata = cachedGroupInfo
			} else {
				groupMetadata, err = i.Client.GetGroupInfo(m.Info.Chat)
				if err != nil {
					i.Log.Error().Err(err).Str("GroupID", m.Info.Chat.String()).Msg("Error getting group metadata")
					return err
				}
				i.SetCachedGroupInfo(groupMetadata)
			}
			for _, participant := range groupMetadata.Participants {
				if participant.JID.User == m.Info.Sender.User && (participant.IsAdmin || participant.IsSuperAdmin) {
					isGroupAdmin = true
					break
				}
				if participant.JID.User == i.Client.Store.ID.User && (participant.IsAdmin || participant.IsSuperAdmin) {
					isBotGroupAdmin = true
					break
				}
			}

			// Group Info
			groupInfo, err = i.UserDB.GetGroupInfo(m.Info.Chat.User)
			if err != nil {
				i.Log.Error().Err(err).Str("Group", m.Info.Chat.String()).Msg("Error getting group from database")
				return err
			}

			// Group Participant Info
			participant, err = i.UserDB.GetParticipant(m.Info.Sender.User, m.Info.Chat.User)
			if err != nil {
				i.Log.Error().Err(err).Str("Group", m.Info.Chat.String()).Str("User", m.Info.Sender.User).Msg("Error getting group participant from database")
				return err
			}

			if isBotGroupAdmin && !isOwner && !isGroupAdmin {
				switch {
				case
					(participant.IsBlacklisted || participant.WarnCount >= 3),
					groupInfo.IsAntiLink && util.MatchURL(messageBody),
					groupInfo.IsAntiWALink && util.MatchWaUrl(messageBody),
					len(tmsg.GetMentionedJIDS(m.Message)) >= len(groupMetadata.Participants)-1:
					if groupInfo.RemoveUser {
						_, err = i.Client.UpdateGroupParticipants(m.Info.Chat, []types.JID{m.Info.Sender}, whatsmeow.ParticipantChangeRemove)
						if err != nil {
							i.Log.Error().Err(err).Str("Group", m.Info.Chat.String()).Str("User", m.Info.Sender.User).Msg("Error removing group participant")
							return err
						}
					}
					i.Client.SendMessage(context.Background(), m.Info.Chat, i.Client.BuildRevoke(m.Info.Chat, m.Info.Sender, m.Info.ID))
					return err
				}
			}

			if !isValid {
				return err
			}

			if isCommand {
				participant.CommandCount++
			} else {
				participant.MessageCount++
			}

			err = i.UserDB.SaveParticipant(participant)
			if err != nil {
				i.Log.Error().Err(err).Str("Group", m.Info.Chat.String()).Str("User", m.Info.Sender.User).Msg("Error updating group participant info")
				return err
			}
		}
		return nil
	}()
	if err != nil {
		return
	}

	// Log info to terminal
	{
		if isCommand {
			if t, ok := i.userLastCommandTime[m.Info.Sender.User]; ok && time.Since(t).Milliseconds() < i.Config.CommandsDelay && !isOwner {
				logFields := i.Log.Info().Str("Command", commandName).Str("User", m.Info.Sender.User)
				if m.Info.IsGroup {
					logFields.Str("Group", groupMetadata.Name)
				}
				logFields.Msg("[SPAM]")
				return
			}
			i.userLastCommandTime[m.Info.Sender.User] = time.Now()
		}

		logFields := i.Log.Info().Str("User", m.Info.Sender.User)
		if isCommand {
			logFields.Str("Command", commandName)
		} else {
			logFields.Str("Message", messageBody)
		}
		if m.Info.IsGroup {
			logFields.Str("Group", groupMetadata.Name)
		}
		logFields.Send()
	}

	if userInfo.IsBanned || (m.Info.IsGroup && !isGroupAdmin && groupInfo.IsBotDisabled) {
		return
	}

	if i.Config.ReadMessages {
		i.Client.MarkRead([]types.MessageID{m.Info.ID}, time.Now(), m.Info.Chat, m.Info.Sender)
	}

	var localizer *i18n.Localizer
	if m.Info.IsGroup {
		localizer = GetLocalizer(userInfo.Language, groupInfo.Language)
	} else {
		localizer = GetLocalizer(userInfo.Language)
	}

	if isCommand {
		ctx := &command.CommandContext{
			Client:    i.Client,
			Config:    i.Config,
			Msg:       m,
			DB:        i.UserDB,
			Body:      messageBody,
			Args:      commandArgs,
			Prefix:    prefix,
			Command:   commandName,
			Localizer: localizer,
			Log:       i.Log,
		}
		cmd, ok := i.cmd.Commands[commandName]
		if ok {
			if cmd.Only.Owner && !isOwner {
				ctx.Reply(ctx.Localizer.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "only.owner",
						Other: "❌ Esse comando só pode ser utilizado pelo meu dono",
					},
				}))
			}

			if cmd.Only.Admin && !isGroupAdmin {
				ctx.Reply(ctx.Localizer.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "only.admin",
						Other: "❌ Esse comando só pode ser utilizado por administradores do grupo",
					},
				}))
			}

			if cmd.Only.Group && !m.Info.IsGroup {
				ctx.Reply(ctx.Localizer.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "only.group",
						Other: "❌ Esse comando só pode ser utilizado em grupos",
					},
				}))
			}

			if cmd.Only.Premium && !userInfo.IsPremium {
				ctx.Reply(ctx.Localizer.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "only.admin",
						Other: "❌ Esse comando só pode ser utilizado por administradores do grupo",
					},
				}))
			}

			if cmd.Need.BotAdmin && !isBotGroupAdmin {
				ctx.Reply(ctx.Localizer.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "need.botadmin",
						Other: "❌ O bot precisa ser administrador para executar esse comando",
					},
				}))
			}

			jid, err := tmsg.GetQuotedJid(m)
			if err != nil {
				i.Log.Error().Err(err).Msg("Error getting quoted jids")
			}

			if cmd.Need.Mention && len(tmsg.GetMentionedJIDS(m.Message)) == 0 && jid.IsEmpty() {
				ctx.Reply(ctx.Localizer.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "need.mention",
						Other: "❌ Você precisa mencionar ou responder a mensagem de alguém",
					},
				}))
			}

			defer func() {
				if r := recover(); r != nil {
					i.Log.Error().Any("Panic", r).Str("Command", commandName).Send()
				}
			}()

			if err := cmd.Run(ctx); err != nil {
				i.Log.Error().Err(err).Str("Command", commandName).Send()
				ctx.Reply(ctx.Localizer.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "error",
						Other: "😵 Ops! Alguma coisa deu errado.",
					},
				}))
			}

		} else if len(commandName) < 14 {
			ma, err := strsim.FindBestMatch(commandName, i.cmd.Aliases)
			if err != nil {
				log.Error().Err(err).Send()
				return
			}
			if ma.BestMatch.Score >= 0.6 {
				ctx.Reply(ctx.Localizer.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "suggestioncommand",
						Other: "⚙️ O comando `{{.Command}}` não foi encontrado. Você quis dizer `{{.Suggestion}}`? Similaridade: {{.Similarity}}%.",
					},
					TemplateData: map[string]any{
						"Command":    commandName,
						"Suggestion": ma.BestMatch.Target,
						"Similarity": fmt.Sprintf("%.0f", ma.BestMatch.Score*100),
					},
				}))
			}
		}
	}
}
