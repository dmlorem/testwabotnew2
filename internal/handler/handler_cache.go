package handler

import (
	"time"

	"go.mau.fi/whatsmeow/types"
)

type cacheEntry struct {
	Info     *types.GroupInfo
	expireAt time.Time
}

func (i *EventHandler) GetCachedGroupInfo(jid types.JID) (*types.GroupInfo, bool) {
	i.groupCacheMutex.Lock()
	defer i.groupCacheMutex.Unlock()
	entry, ok := i.groupInfoCache[jid.User]

	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expireAt) {
		delete(i.groupInfoCache, jid.User)
		return nil, false
	}

	if entry.Info == nil {
		return nil, false
	}

	return entry.Info, true
}

func (i *EventHandler) SetCachedGroupInfo(group *types.GroupInfo) {
	if group == nil {
		return
	}
	i.groupCacheMutex.Lock()
	i.groupInfoCache[group.JID.User] = &cacheEntry{
		Info:     group,
		expireAt: time.Now().Add(6 * time.Hour),
	}
	i.groupCacheMutex.Unlock()
}
