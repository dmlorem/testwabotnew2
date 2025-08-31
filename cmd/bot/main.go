package main

import (
	"meowabot/internal/app"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

func main() {
	configPath := "./config/config.toml"
	sessionPath := "./data/database.db"
	databasePath := "./data/session.db"
	logger := zerolog.
		New(os.Stdout).
		With().
		Timestamp().
		Logger().
		Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.TimeOnly,
			FormatMessage: func(s any) string {
				if s, ok := s.(string); ok {
					return s
				}
				return ""
			},
		}).
		Level(zerolog.InfoLevel)

	meow, err := app.StartMeowbot(configPath, sessionPath, databasePath, &logger)
	if err != nil {
		panic(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGABRT)
	<-c

	meow.Client.Disconnect()
	meow.Container.Close()
	meow.UserDB.Close()
}
