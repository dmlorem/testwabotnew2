package database

import (
	"sync"

	"gorm.io/gorm"
)

type DBInstance struct {
	db *gorm.DB
	MU sync.RWMutex
}

func NewDB(dialector gorm.Dialector, config *gorm.Config) (*DBInstance, error) {
	db, err := gorm.Open(dialector, config)
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(
		&User{},
		&Group{},
		&GroupParticipant{},
		&Feed{},
		&FeedSubscriptions{},
		
	)
	if err != nil {
		return nil, err
	}
	return &DBInstance{db: db}, nil
}

func (d *DBInstance) Close() error {
	db, err := d.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}
