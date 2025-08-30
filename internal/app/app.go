package app

import (
	"context"
	"fmt"
	"meowabot/internal/config"
	"meowabot/internal/database"
	"meowabot/internal/handler"
	"time"

	_ "meowabot/internal/app/commands"

	"github.com/rs/zerolog"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func StartMeowbot(configPath string, sessionPath string, databasePath string, logger *zerolog.Logger) (*handler.EventHandler, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	container, err := sqlstore.New(ctx, "sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", sessionPath), waLog.Noop)
	if err != nil {
		return nil, err
	}

	db, err := database.NewDB(sqlite.Open(fmt.Sprintf("%s?_foreign_keys=on", databasePath)), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		return nil, err
	}

	config, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, err
	}

	cli := whatsmeow.NewClient(deviceStore, waLog.Noop)

	opts := handler.EventHandlerOptions{
		Config:    config,
		Client:    cli,
		Container: container,
		UserDB:    db,
		Logger:    logger,
		WaLogger:  waLog.Zerolog(logger.With().Str("Source", "Client").Logger()),
	}

	evthandler := handler.NewEventHandler(opts)
	err = evthandler.Connect(ctx)
	if err != nil {
		return nil, err
	}

	func() {
		<-evthandler.WaitAuthenticate()
		groups, err := evthandler.Client.GetJoinedGroups()
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get joined groups")
			return
		}
		for _, group := range groups {
			evthandler.SetCachedGroupInfo(group)
			participants := make([]string, len(group.Participants))
			for i, p := range group.Participants {
				participants[i] = p.JID.User
			}
			err = db.UpdateGroupParticipants(group.JID.User, participants)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to update group participants")
				return
			}
		}
	}()

	return evthandler, nil
}
