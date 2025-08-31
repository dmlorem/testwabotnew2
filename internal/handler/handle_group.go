package handler

import (
	"slices"
	"sort"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (i *EventHandler) handleGroupInfoChange(event *events.GroupInfo) {
	i.groupCacheMutex.Lock()
	defer i.groupCacheMutex.Unlock()

	groupMetadata, ok := i.groupInfoCache[event.JID.User]
	if !ok {
		return
	}
	if time.Now().After(groupMetadata.expireAt) {
		delete(i.groupInfoCache, event.JID.User)
		return
	}

	var isBotGroupAdmin bool
	for _, p := range groupMetadata.Info.Participants {
		if p.JID.User == i.Client.Store.ID.User && (p.IsAdmin || p.IsSuperAdmin) {
			isBotGroupAdmin = true
			break
		}
	}

	groupInfo, err := i.UserDB.GetGroupInfo(event.JID.User)
	if err != nil {
		i.Log.Error().Err(err).Str("GroupID", event.JID.User).Msg("Error retrieving group info from database")
		return
	}

	switch {
	case event.Name != nil:
		groupMetadata.Info.GroupName = *event.Name
	case event.Topic != nil:
		groupMetadata.Info.GroupTopic = *event.Topic
	case event.Locked != nil:
		groupMetadata.Info.GroupLocked = *event.Locked
	case event.Announce != nil:
		groupMetadata.Info.IsAnnounce = event.Announce.IsAnnounce
		groupMetadata.Info.AnnounceVersionID = event.Announce.AnnounceVersionID
	case event.Ephemeral != nil:
		groupMetadata.Info.GroupEphemeral = *event.Ephemeral
	case event.MembershipApprovalMode != nil:
		groupMetadata.Info.GroupMembershipApprovalMode = *event.MembershipApprovalMode
	}

	participantMap := make(map[string]int, len(groupMetadata.Info.Participants))

	for i, v := range groupMetadata.Info.Participants {
		participantMap[v.JID.User] = i
	}

	switch {
	case len(event.Leave) > 0:
		participantsToRemove := []int{}
		for _, user := range event.Leave {
			if index, found := participantMap[user.User]; found {
				participantsToRemove = append(participantsToRemove, index)
			}
			if user.User == i.Client.Store.ID.User {
				delete(i.groupInfoCache, event.JID.User)
				if err := i.UserDB.DeleteGroupInfo(groupInfo); err != nil {
					i.Log.Error().Err(err).Str("GroupID", event.JID.String()).Msg("Error deleting row from group info")
				}
				return
			}
		}
		sort.Sort(sort.Reverse(sort.IntSlice(participantsToRemove)))
		i.UserDB.MU.Lock()
		for _, u := range participantsToRemove {
			userInfo, err := i.UserDB.GetParticipant(groupMetadata.Info.Participants[u].JID.User, groupMetadata.Info.JID.User)
			if err != nil {
				i.Log.Error().Err(err).Str("GroupID", groupMetadata.Info.JID.User).Str("User", groupMetadata.Info.Participants[u].JID.User).Msg("Error retrieving user from database")
				continue
			}
			if err = i.UserDB.DeleteParticipant(userInfo); err != nil {
				i.Log.Error().Err(err).Str("GroupID", groupMetadata.Info.JID.User).Str("User", groupMetadata.Info.Participants[u].JID.User).Msg("Error deleting user from database")
			}
			groupMetadata.Info.Participants = slices.Delete(groupMetadata.Info.Participants, u, u+1)
		}
		i.UserDB.MU.Unlock()

	case len(event.Join) > 0:
		for _, user := range event.Join {
			if groupInfo.AllowedDDIS != "" && isBotGroupAdmin && event.Sender == nil { // TODO: Change to isGroupAdmin == false
				var valid bool
				ddis := strings.SplitSeq(groupInfo.AllowedDDIS, ",")
				for ddi := range ddis {
					if strings.HasPrefix(user.User, ddi) {
						valid = true
						break
					}
				}
				if !valid {
					if _, err := i.Client.UpdateGroupParticipants(event.JID, []types.JID{user}, whatsmeow.ParticipantChangeRemove); err != nil {
						i.Log.Error().Str("ChatID", event.JID.String()).Str("UserID", user.String()).Msg("Error removing user")
					}
					continue
				}
			}
			if userInfo, err := i.UserDB.GetParticipant(user.User, event.JID.User); err != nil && userInfo.IsBlacklisted {
				if isBotGroupAdmin {
					if _, err := i.Client.UpdateGroupParticipants(event.JID, []types.JID{user}, whatsmeow.ParticipantChangeRemove); err != nil {
						i.Log.Error().Str("ChatID", event.JID.String()).Str("UserID", user.String()).Msg("Error removing blacklisted user")
					}
					continue
				}
			}
			if _, found := participantMap[user.User]; !found {
				groupMetadata.Info.Participants = append(groupMetadata.Info.Participants, types.GroupParticipant{JID: user, IsAdmin: false, IsSuperAdmin: false})
			}
		}

	case len(event.Promote) > 0:
		for _, user := range event.Promote {
			if index, found := participantMap[user.User]; found {
				groupMetadata.Info.Participants[index].IsAdmin = true
			}
		}

	case len(event.Demote) > 0:
		for _, user := range event.Demote {
			if index, found := participantMap[user.User]; found {
				groupMetadata.Info.Participants[index].IsAdmin = false
			}
		}
	}

	i.groupInfoCache[event.JID.User] = groupMetadata
}
