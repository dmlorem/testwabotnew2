package handler

import (
	"meowabot/internal/command"
	"meowabot/internal/config"
	"meowabot/internal/database"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type EventHandler struct {
	Config    *config.ConfigScheme
	Client    *whatsmeow.Client
	Container *sqlstore.Container
	UserDB    *database.DBInstance
	Log       *zerolog.Logger
	WaLogger  waLog.Logger

	cmd                 *command.CommandList
	pairedChannel       []chan<- error
	authChannel         []chan<- struct{}
	logoutChannel       []chan<- struct{}
	receivedOldEvents   atomic.Bool
	wg                  sync.WaitGroup
	groupInfoCache      map[string]*cacheEntry
	groupCacheMutex     sync.Mutex
	userLastCommandTime map[string]time.Time
}

type EventHandlerOptions struct {
	Config    *config.ConfigScheme
	Client    *whatsmeow.Client
	Container *sqlstore.Container
	UserDB    *database.DBInstance
	Logger    *zerolog.Logger
	WaLogger  waLog.Logger
}

func NewEventHandler(opts EventHandlerOptions) *EventHandler {
	evt := &EventHandler{
		Config:    opts.Config,
		Client:    opts.Client,
		Container: opts.Container,
		UserDB:    opts.UserDB,
		Log:       opts.Logger,
		WaLogger:  opts.WaLogger,

		cmd:                 command.Default,
		groupInfoCache:      make(map[string]*cacheEntry),
		userLastCommandTime: make(map[string]time.Time),
	}
	evt.receivedOldEvents.Store(true)
	opts.Client.AddEventHandler(evt.handleEvent)
	return evt
}

func (i *EventHandler) handleEvent(evt any) {
	switch event := evt.(type) {
	case *events.Message:
		i.wg.Add(1)
		go func(event *events.Message) {
			i.handleMessage(event)
			i.wg.Done()
		}(event)
	case *events.GroupInfo:
		go i.handleGroupInfoChange(event)
	case *events.CallOffer:
		if err := i.Client.RejectCall(event.From, event.CallID); err != nil {
			log.Error().Err(err).Msg("Error rejecting call")
		}
	case *events.OfflineSyncPreview:
		i.receivedOldEvents.Store(false)
		log.Info().
			Str("AppDataChanges", strconv.Itoa(event.AppDataChanges)).
			Str("Messages", strconv.Itoa(event.Messages)).
			Str("Notifications", strconv.Itoa(event.Notifications)).
			Str("Receipts", strconv.Itoa(event.Receipts)).
			Msg("Receiving old events")
	case *events.OfflineSyncCompleted:
		i.wg.Wait()
		if !i.receivedOldEvents.Swap(true) {
			log.Info().Msg("All old events received")
		}
	case *events.Connected:
		for _, ch := range i.authChannel {
			ch <- struct{}{}
			close(ch)
		}
		clear(i.authChannel)
	case *events.PairSuccess:
		for _, ch := range i.pairedChannel {
			ch <- nil
			close(ch)
		}
		clear(i.pairedChannel)
	case *events.PairError:
		for _, ch := range i.pairedChannel {
			ch <- event.Error
			close(ch)
		}
		clear(i.pairedChannel)
	case *events.LoggedOut:
		for _, ch := range i.logoutChannel {
			ch <- struct{}{}
			close(ch)
		}
		clear(i.logoutChannel)
	}
}
