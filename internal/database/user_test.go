package database

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetUserInfo(t *testing.T) {
	db := setupTestDB(t)
	userID := "testuser1"

	// First call: should create user
	user, err := db.GetUserInfo(userID)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, userID, user.ID)

	// Second call: should retrieve same user
	user2, err := db.GetUserInfo(userID)
	require.NoError(t, err)
	require.NotNil(t, user2)
	require.Equal(t, userID, user2.ID)
	require.Equal(t, user.ID, user2.ID)
}

func TestSaveUserInfo(t *testing.T) {
	db := setupTestDB(t)
	userID := "testuser2"

	user, err := db.GetUserInfo(userID)
	require.NoError(t, err)
	user.Name = "Alice"

	err = db.SaveUserInfo(user)
	require.NoError(t, err)

	// Retrieve again and check fields
	user2, err := db.GetUserInfo(userID)
	require.NoError(t, err)
	require.Equal(t, "Alice", user2.Name)
}
