package database

import (
	"errors"

	"gorm.io/gorm"
)

func (d *DBInstance) GetGroupInfo(groupID string) (*Group, error) {
	var group Group
	result := d.db.First(&group, &Group{ID: groupID})
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			newGroup := Group{ID: groupID}
			if err := d.db.Create(&newGroup).Error; err != nil {
				return nil, err
			}
			return &newGroup, nil
		}
		return nil, result.Error
	}
	return &group, nil
}

func (d *DBInstance) SaveGroupInfo(groupInfo *Group) error {
	return d.db.Save(groupInfo).Error
}

func (d *DBInstance) DeleteGroupInfo(groupInfo *Group) error {
	return d.db.Delete(groupInfo).Error
}
