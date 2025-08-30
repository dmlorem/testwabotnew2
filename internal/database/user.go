package database

func (d *DBInstance) GetUserInfo(userID string) (*User, error) {
	var user User
	result := d.db.Where(&User{ID: userID}).FirstOrCreate(&user)

	if result.Error != nil {
		return nil, result.Error
	}

	return &user, nil
}

func (d *DBInstance) SaveUserInfo(userInfo *User) error {
	return d.db.Save(userInfo).Error
}
