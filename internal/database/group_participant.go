package database

import "gorm.io/gorm"

func (d *DBInstance) GetParticipant(userID string, groupID string) (*GroupParticipant, error) {
	var participant = GroupParticipant{}
	err := d.db.Where(&GroupParticipant{GroupID: groupID, UserID: userID}).FirstOrCreate(&participant).Error
	if err != nil {
		return nil, err
	}
	return &participant, nil
}

func (d *DBInstance) SaveParticipant(member *GroupParticipant) error {
	return d.db.Save(member).Error
}

func (d *DBInstance) DeleteParticipant(member *GroupParticipant) error {
	return d.db.Where(member).Delete(&GroupParticipant{}).Error
}

func (d *DBInstance) GetAllParticipants(groupID string) ([]GroupParticipant, error) {
	var members = []GroupParticipant{}
	err := d.db.Where(&GroupParticipant{GroupID: groupID}).Find(&members).Error
	return members, err
}

func (d *DBInstance) UpdateGroupParticipants(groupID string, participantIDs []string) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		group := Group{ID: groupID}
		if err := tx.FirstOrCreate(&group, Group{ID: groupID}).Error; err != nil {
			return err
		}

		var existingParticipants []GroupParticipant
		if err := tx.Where("group_id = ?", groupID).Find(&existingParticipants).Error; err != nil {
			return err
		}

		existingMap := make(map[string]struct{}, len(existingParticipants))
		for _, p := range existingParticipants {
			existingMap[p.UserID] = struct{}{}
		}

		desiredMap := make(map[string]struct{}, len(participantIDs))
		for _, id := range participantIDs {
			desiredMap[id] = struct{}{}
		}

		var toDelete []string
		for id := range existingMap {
			if _, ok := desiredMap[id]; !ok {
				toDelete = append(toDelete, id)
			}
		}

		if len(toDelete) > 0 {
			if err := tx.Where("group_id = ? AND user_id IN ?", groupID, toDelete).Delete(&GroupParticipant{}).Error; err != nil {
				return err
			}
		}

		var toInsert []GroupParticipant
		for _, id := range participantIDs {
			if _, ok := existingMap[id]; !ok {
				toInsert = append(toInsert, GroupParticipant{
					GroupID: groupID,
					UserID:  id,
				})
			}
		}

		if len(toInsert) > 0 {
			if err := tx.Create(&toInsert).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
