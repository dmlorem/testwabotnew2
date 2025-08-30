package database

import (
	"time"
)

type User struct {
	ID                 string `gorm:"primaryKey"`
	Name               string `gorm:"default:'';not null"`
	CommandCount       uint32 `gorm:"default:0;not null"`
	IsBanned           bool   `gorm:"default:false;not null"`
	IsPremium          bool   `gorm:"default:false;not null"`
	Language           string `gorm:"default:'';not null"`
	StickerDescription string `gorm:"default:'';not null"`
	StickerTitle       string `gorm:"default:'';not null"`
}

type Group struct {
	ID                string `gorm:"column:id;primaryKey"`
	AllowedDDIS       string `gorm:"default:'';not null"`
	AutoDownloadMedia bool   `gorm:"default:false;not null"`
	IsAntiLink        bool   `gorm:"default:false;not null"`
	IsAntiWALink      bool   `gorm:"default:false;not null"`
	IsBotDisabled     bool   `gorm:"default:false;not null"`
	Language          string `gorm:"default:'';not null"`
	RemoveUser        bool   `gorm:"default:true;not null"`
}

type GroupParticipant struct {
	GroupID       string `gorm:"column:group_id;primaryKey"`
	UserID        string `gorm:"column:user_id;primaryKey"`
	MessageCount  uint64 `gorm:"default:0;not null"`
	CommandCount  uint64 `gorm:"default:0;not null"`
	WarnCount     uint8  `gorm:"default:0;not null"`
	IsBlacklisted bool   `gorm:"default:false;not null"`

	Group Group `gorm:"foreignKey:GroupID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	User  User  `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type Feed struct {
	ID          uint32    `gorm:"primarykey;autoIncrement"`
	URL         string    `gorm:"not null"`
	LastUpdated time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}

type FeedSubscriptions struct {
	GroupID string `gorm:"primaryKey"`
	FeedID  string `gorm:"primaryKey"`

	Feed  Feed  `gorm:"foreignKey:FeedID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Group Group `gorm:"foreignKey:GroupID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
