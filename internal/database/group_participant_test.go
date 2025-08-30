//go:build !integration

package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetParticipant(t *testing.T) {
	db := setupTestDB(t)
	userID := "user1"
	groupID := "group1"

	// 1. Test creating a new participant
	participant, err := db.GetParticipant(userID, groupID)
	require.NoError(t, err)
	require.NotNil(t, participant)
	assert.Equal(t, userID, participant.UserID)
	assert.Equal(t, groupID, participant.GroupID)

	// 2. Test retrieving an existing participant
	participant2, err := db.GetParticipant(userID, groupID)
	require.NoError(t, err)
	require.NotNil(t, participant2)
	assert.Equal(t, userID, participant2.UserID)
	assert.Equal(t, groupID, participant2.GroupID)
}

func TestSaveParticipant(t *testing.T) {
	db := setupTestDB(t)
	userID := "user2"
	groupID := "group2"

	// Get or create participant
	participant, err := db.GetParticipant(userID, groupID)
	require.NoError(t, err)
	require.NotNil(t, participant)

	// Modify and save
	participant.WarnCount = 3
	participant.IsBlacklisted = true
	err = db.SaveParticipant(participant)
	require.NoError(t, err)

	// Retrieve again and verify changes
	participant2, err := db.GetParticipant(userID, groupID)
	require.NoError(t, err)
	require.NotNil(t, participant2)
	assert.Equal(t, participant2.WarnCount, uint8(3))
	assert.Equal(t, participant2.IsBlacklisted, true)

	// Modify again
	participant2.WarnCount = 2
	participant2.IsBlacklisted = false
	err = db.SaveParticipant(participant2)
	require.NoError(t, err)

	// Retrieve and verify
	participant3, err := db.GetParticipant(userID, groupID)
	require.NoError(t, err)
	require.NotNil(t, participant3)
	assert.False(t, participant3.IsBlacklisted)
	assert.Equal(t, participant3.WarnCount, uint8(2))
}

func TestDeleteParticipant(t *testing.T) {
	db := setupTestDB(t)
	userID := "user3"
	groupID := "group3"

	// Create participant
	participant, err := db.GetParticipant(userID, groupID)
	require.NoError(t, err)
	require.NotNil(t, participant)

	// Delete participant
	err = db.DeleteParticipant(participant)
	require.NoError(t, err)

	// Try to get all participants for the group - should be empty
	allParticipants, err := db.GetAllParticipants(groupID)
	require.NoError(t, err)
	assert.Empty(t, allParticipants, "Group should have no participants after deletion")

	// GetParticipant uses FirstOrCreate, so calling it again will recreate the participant
	participant2, err := db.GetParticipant(userID, groupID)
	require.NoError(t, err)
	require.NotNil(t, participant2)
	assert.Equal(t, userID, participant2.UserID)
	assert.Equal(t, groupID, participant2.GroupID)
}

func TestGetAllParticipants(t *testing.T) {
	db := setupTestDB(t)
	groupID := "group4"
	userIDs := []string{"user4_1", "user4_2", "user4_3"}

	// Check initially empty
	participants, err := db.GetAllParticipants(groupID)
	require.NoError(t, err)
	assert.Empty(t, participants)

	// Create participants
	for _, userID := range userIDs {
		_, err := db.GetParticipant(userID, groupID)
		require.NoError(t, err)
	}

	// Get all participants
	participants, err = db.GetAllParticipants(groupID)
	require.NoError(t, err)
	require.Len(t, participants, len(userIDs), "Should retrieve all created participants")

	// Verify user IDs are present
	retrievedUserIDs := make(map[string]struct{})
	for _, p := range participants {
		assert.Equal(t, groupID, p.GroupID)
		retrievedUserIDs[p.UserID] = struct{}{}
	}
	for _, userID := range userIDs {
		_, found := retrievedUserIDs[userID]
		assert.True(t, found, "Expected user ID %s not found", userID)
	}

	// Check another group is empty
	participants, err = db.GetAllParticipants("other_group")
	require.NoError(t, err)
	assert.Empty(t, participants)
}

func TestUpdateGroupParticipants(t *testing.T) {
	db := setupTestDB(t)
	groupID := "group5"

	// --- Test Case 1: Initial creation ---
	initialParticipants := []string{"userA", "userB"}
	err := db.UpdateGroupParticipants(groupID, initialParticipants)
	require.NoError(t, err)

	// Verify participants exist
	participants, err := db.GetAllParticipants(groupID)
	require.NoError(t, err)
	require.Len(t, participants, 2)
	participantMap := make(map[string]struct{})
	for _, p := range participants {
		participantMap[p.UserID] = struct{}{}
	}
	assert.Contains(t, participantMap, "userA")
	assert.Contains(t, participantMap, "userB")

	// --- Test Case 2: Add new participants ---
	addParticipants := []string{"userA", "userB", "userC"}
	err = db.UpdateGroupParticipants(groupID, addParticipants)
	require.NoError(t, err)

	participants, err = db.GetAllParticipants(groupID)
	require.NoError(t, err)
	require.Len(t, participants, 3)
	participantMap = make(map[string]struct{})
	for _, p := range participants {
		participantMap[p.UserID] = struct{}{}
	}
	assert.Contains(t, participantMap, "userA")
	assert.Contains(t, participantMap, "userB")
	assert.Contains(t, participantMap, "userC")

	// --- Test Case 3: Remove participants ---
	removeParticipants := []string{"userC"}
	err = db.UpdateGroupParticipants(groupID, removeParticipants)
	require.NoError(t, err)

	participants, err = db.GetAllParticipants(groupID)
	require.NoError(t, err)
	require.Len(t, participants, 1)
	assert.Equal(t, "userC", participants[0].UserID)

	// --- Test Case 4: Add and Remove simultaneously ---
	addRemoveParticipants := []string{"userC", "userD"}
	err = db.UpdateGroupParticipants(groupID, addRemoveParticipants)
	require.NoError(t, err)

	participants, err = db.GetAllParticipants(groupID)
	require.NoError(t, err)
	require.Len(t, participants, 2)
	participantMap = make(map[string]struct{})
	for _, p := range participants {
		participantMap[p.UserID] = struct{}{}
	}
	assert.Contains(t, participantMap, "userC")
	assert.Contains(t, participantMap, "userD")

	// --- Test Case 5: Remove all participants ---
	removeAllParticipants := []string{}
	err = db.UpdateGroupParticipants(groupID, removeAllParticipants)
	require.NoError(t, err)

	participants, err = db.GetAllParticipants(groupID)
	require.NoError(t, err)
	assert.Empty(t, participants)

	// --- Test Case 6: No changes ---
	err = db.UpdateGroupParticipants(groupID, removeAllParticipants) // Still empty
	require.NoError(t, err)

	participants, err = db.GetAllParticipants(groupID)
	require.NoError(t, err)
	assert.Empty(t, participants)

	// --- Test Case 7: Add back after empty ---
	addBackParticipants := []string{"userE"}
	err = db.UpdateGroupParticipants(groupID, addBackParticipants)
	require.NoError(t, err)

	participants, err = db.GetAllParticipants(groupID)
	require.NoError(t, err)
	require.Len(t, participants, 1)
	assert.Equal(t, "userE", participants[0].UserID)
}
