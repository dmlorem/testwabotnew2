//go:build !integration

package database

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetGroupInfo(t *testing.T) {
	db := setupTestDB(t)
	groupID := "group1"

	// Should create group if not exists
	group, err := db.GetGroupInfo(groupID)
	require.NoError(t, err)
	require.NotNil(t, group)
	require.Equal(t, groupID, group.ID)

	// Should retrieve same group
	group2, err := db.GetGroupInfo(groupID)
	require.NoError(t, err)
	require.NotNil(t, group2)
	require.Equal(t, groupID, group2.ID)
	require.Equal(t, group.ID, group2.ID)
}

func TestSaveGroupInfo(t *testing.T) {
	db := setupTestDB(t)
	groupID := "group2"

	group, err := db.GetGroupInfo(groupID)
	require.NoError(t, err)
	group.Language = "en"

	err = db.SaveGroupInfo(group)
	require.NoError(t, err)

	// Retrieve again and check field
	group2, err := db.GetGroupInfo(groupID)
	require.NoError(t, err)
	require.Equal(t, "en", group2.Language)
}

func TestDeleteGroupInfo(t *testing.T) {
	db := setupTestDB(t)
	groupID := "group3"

	group, err := db.GetGroupInfo(groupID)
	require.NoError(t, err)

	err = db.DeleteGroupInfo(group)
	require.NoError(t, err)

	// Should recreate on next GetGroupInfo
	group2, err := db.GetGroupInfo(groupID)
	require.NoError(t, err)
	require.NotNil(t, group2)
	require.Equal(t, groupID, group2.ID)
}
func TestGetGroupInfo_CreatesGroupIfNotExists(t *testing.T) {
	db := setupTestDB(t)
	groupID := "testgroup_create"

	group, err := db.GetGroupInfo(groupID)
	require.NoError(t, err)
	require.NotNil(t, group)
	require.Equal(t, groupID, group.ID)
}

func TestGetGroupInfo_ReturnsExistingGroup(t *testing.T) {
	db := setupTestDB(t)
	groupID := "testgroup_existing"

	// Create group first
	created, err := db.GetGroupInfo(groupID)
	require.NoError(t, err)
	require.NotNil(t, created)

	// Retrieve again
	retrieved, err := db.GetGroupInfo(groupID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, created.ID, retrieved.ID)
}

func TestGetGroupInfo_ErrorOnDBFailure(t *testing.T) {
	db := setupTestDB(t)
	groupID := "testgroup_error"

	// Simulate DB error by closing DB
	sqlDB, _ := db.db.DB()
	sqlDB.Close()

	group, err := db.GetGroupInfo(groupID)
	require.Error(t, err)
	require.Nil(t, group)
}
