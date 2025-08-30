package handler

import (
	"context"
	"fmt"
	"meowabot/internal/database"
	tmsg "meowabot/internal/tools/messages"
	"meowabot/internal/util"
	"slices"
	"strings"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
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
	var command string
	var commandArgs string
	fmt.Println(commandArgs)
	if isCommand {
		command = util.NormalizeString(strings.ToLower(strings.Split(strings.TrimSpace(strings.TrimPrefix(messageBody, prefix)), " ")[0]))
		commandArgs = strings.TrimSpace(messageBody[len(command)+len(prefix):])
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
			i.Logger.Error().Err(err).Str("User", m.Info.Sender.User).Msg("Error retrieving user from database")
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
			i.Logger.Error().Err(err).Str("User", m.Info.Sender.User).Msg("Error saving user info")
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
					i.Logger.Error().Err(err).Str("GroupID", m.Info.Chat.String()).Msg("Error getting group metadata")
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
				i.Logger.Error().Err(err).Str("Group", m.Info.Chat.String()).Msg("Error getting group from database")
				return err
			}

			// Group Participant Info
			participant, err = i.UserDB.GetParticipant(m.Info.Sender.User, m.Info.Chat.User)
			if err != nil {
				i.Logger.Error().Err(err).Str("Group", m.Info.Chat.String()).Str("User", m.Info.Sender.User).Msg("Error getting group participant from database")
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
							i.Logger.Error().Err(err).Str("Group", m.Info.Chat.String()).Str("User", m.Info.Sender.User).Msg("Error removing group participant")
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
				i.Logger.Error().Err(err).Str("Group", m.Info.Chat.String()).Str("User", m.Info.Sender.User).Msg("Error updating group participant info")
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
				logFields := i.Logger.Info().Str("Command", command).Str("User", m.Info.Sender.User)
				if m.Info.IsGroup {
					logFields.Str("Group", groupMetadata.Name)
				}
				logFields.Msg("[SPAM]")
				return
			}
			i.userLastCommandTime[m.Info.Sender.User] = time.Now()
		}

		logFields := i.Logger.Info().Str("User", m.Info.Sender.User)
		if isCommand {
			logFields.Str("Command", command)
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

	fmt.Println(localizer == nil)

}
