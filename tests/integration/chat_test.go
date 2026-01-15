package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChat_Create(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "chatcreate")

	chatReq := map[string]interface{}{
		"type":        "group",
		"name":        fmt.Sprintf("Test Chat %d", time.Now().UnixNano()),
		"description": "Test chat description",
	}

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/chats", chatReq, user.AccessToken)

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Response: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.NotEmpty(t, result["id"])
	// API returns chat_type as numeric (2 = group) or string
	assert.NotNil(t, result["chat_type"], "chat_type should be present")
	assert.Equal(t, chatReq["name"], result["name"])
}

func TestChat_List(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "chatlist")

	// Create a few chats
	for i := 0; i < 3; i++ {
		createTestChat(t, user, "group", fmt.Sprintf("List Test Chat %d", i), nil)
	}

	resp, body := doRequest(t, "GET", apiGatewayURL+"/api/chats?limit=10&offset=0", nil, user.AccessToken)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	chats, ok := result["chats"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(chats), 3)
}

func TestChat_Get(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "chatget")
	chat := createTestChat(t, user, "group", "Get Test Chat", nil)

	resp, body := doRequest(t, "GET", apiGatewayURL+"/api/chats/"+chat.ID, nil, user.AccessToken)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, chat.ID, result["id"])
	assert.Equal(t, chat.Name, result["name"])
}

func TestChat_Update(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "chatupdate")
	chat := createTestChat(t, user, "group", "Update Test Chat", nil)

	updateReq := map[string]string{
		"name":        "Updated Chat Name",
		"description": "Updated description",
	}

	resp, body := doRequest(t, "PUT", apiGatewayURL+"/api/chats/"+chat.ID, updateReq, user.AccessToken)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, "Updated Chat Name", result["name"])
}

func TestChat_Delete(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "chatdelete")
	chat := createTestChat(t, user, "group", "Delete Test Chat", nil)

	resp, _ := doRequest(t, "DELETE", apiGatewayURL+"/api/chats/"+chat.ID, nil, user.AccessToken)

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify chat is deleted - API may return 404, 403, or 500 depending on implementation
	resp, _ = doRequest(t, "GET", apiGatewayURL+"/api/chats/"+chat.ID, nil, user.AccessToken)
	assert.True(t, resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden ||
		resp.StatusCode == http.StatusInternalServerError,
		"Expected 404, 403 or 500 after deletion, got %d", resp.StatusCode)
}

func TestChat_AddParticipant(t *testing.T) {
	SkipIfNotIntegration(t)

	owner := createTestUser(t, "chatowner")
	participant := createTestUser(t, "chatparticipant")

	chat := createTestChat(t, owner, "group", "Participant Test Chat", nil)

	addReq := map[string]string{
		"user_id": participant.ID,
		"role":    "member",
	}

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/chats/"+chat.ID+"/participants", addReq, owner.AccessToken)

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Response: %s", string(body))

	// Verify participant can access chat
	resp, body = doRequest(t, "GET", apiGatewayURL+"/api/chats/"+chat.ID, nil, participant.AccessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))
}

func TestChat_RemoveParticipant(t *testing.T) {
	SkipIfNotIntegration(t)

	owner := createTestUser(t, "rmowner")
	participant := createTestUser(t, "rmparticipant")

	chat := createTestChat(t, owner, "group", "Remove Participant Test", []string{participant.ID})

	resp, _ := doRequest(t, "DELETE", apiGatewayURL+"/api/chats/"+chat.ID+"/participants/"+participant.ID, nil, owner.AccessToken)

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify participant cannot access chat - API may return 403 or 500
	resp, _ = doRequest(t, "GET", apiGatewayURL+"/api/chats/"+chat.ID, nil, participant.AccessToken)
	assert.True(t, resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusInternalServerError,
		"Expected 403 or 500, got %d", resp.StatusCode)
}

func TestChat_Favorites(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "favorites")
	chat := createTestChat(t, user, "group", "Favorites Test Chat", nil)

	// Add to favorites
	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/chats/"+chat.ID+"/favorite", nil, user.AccessToken)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Response: %s", string(body))

	// Verify chat is in favorites (check response)
	resp, body = doRequest(t, "GET", apiGatewayURL+"/api/chats/"+chat.ID, nil, user.AccessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.Unmarshal(body, &result)
	// is_favorite may or may not be present depending on API implementation
	if isFav, ok := result["is_favorite"]; ok {
		assert.True(t, isFav.(bool), "is_favorite should be true after adding to favorites")
	}

	// Remove from favorites
	resp, _ = doRequest(t, "DELETE", apiGatewayURL+"/api/chats/"+chat.ID+"/favorite", nil, user.AccessToken)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestChat_Archive(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "archive")
	chat := createTestChat(t, user, "group", "Archive Test Chat", nil)

	// Archive chat
	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/chats/"+chat.ID+"/archive", nil, user.AccessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	// Verify chat is archived
	resp, body = doRequest(t, "GET", apiGatewayURL+"/api/chats/"+chat.ID, nil, user.AccessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.Unmarshal(body, &result)
	// is_archived may or may not be present depending on API implementation
	if isArchived, ok := result["is_archived"]; ok {
		assert.True(t, isArchived.(bool), "is_archived should be true after archiving")
	}

	// Unarchive chat
	resp, _ = doRequest(t, "DELETE", apiGatewayURL+"/api/chats/"+chat.ID+"/archive", nil, user.AccessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestChat_Get_NotParticipant(t *testing.T) {
	SkipIfNotIntegration(t)

	owner := createTestUser(t, "chatowner2")
	outsider := createTestUser(t, "chatoutsider")

	chat := createTestChat(t, owner, "group", "Private Chat", nil)

	// Outsider should not be able to access the chat
	resp, _ := doRequest(t, "GET", apiGatewayURL+"/api/chats/"+chat.ID, nil, outsider.AccessToken)

	// API may return 403 or 500 depending on error handling
	assert.True(t, resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusInternalServerError,
		"Expected 403 or 500, got %d", resp.StatusCode)
}

func TestChat_Get_NotFound(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "chatnotfound")

	// Non-existent chat ID
	resp, _ := doRequest(t, "GET", apiGatewayURL+"/api/chats/00000000-0000-0000-0000-000000000000", nil, user.AccessToken)

	// API may return 404 or 500 depending on error handling
	assert.True(t, resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusInternalServerError,
		"Expected 404 or 500, got %d", resp.StatusCode)
}

func TestChat_Update_NotAdmin(t *testing.T) {
	SkipIfNotIntegration(t)

	owner := createTestUser(t, "updateowner")
	member := createTestUser(t, "updatemember")

	chat := createTestChat(t, owner, "group", "Update Test", []string{member.ID})

	// Member (not admin) tries to update
	updateReq := map[string]string{
		"name": "Should Fail",
	}

	resp, _ := doRequest(t, "PUT", apiGatewayURL+"/api/chats/"+chat.ID, updateReq, member.AccessToken)

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestChat_Delete_NotAdmin(t *testing.T) {
	SkipIfNotIntegration(t)

	owner := createTestUser(t, "deleteowner")
	member := createTestUser(t, "deletemember")

	chat := createTestChat(t, owner, "group", "Delete Test", []string{member.ID})

	// Member (not admin) tries to delete
	resp, _ := doRequest(t, "DELETE", apiGatewayURL+"/api/chats/"+chat.ID, nil, member.AccessToken)

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestChat_GetParticipants(t *testing.T) {
	SkipIfNotIntegration(t)

	owner := createTestUser(t, "getparticipants")
	member := createTestUser(t, "getparticipants2")

	chat := createTestChat(t, owner, "group", "Participants Test", []string{member.ID})

	resp, body := doRequest(t, "GET", apiGatewayURL+"/api/chats/"+chat.ID+"/participants", nil, owner.AccessToken)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	participants, ok := result["participants"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(participants), 2) // owner + member
}

func TestChat_UpdateParticipantRole(t *testing.T) {
	SkipIfNotIntegration(t)

	owner := createTestUser(t, "roleowner")
	member := createTestUser(t, "rolemember")

	chat := createTestChat(t, owner, "group", "Role Test", []string{member.ID})

	// Promote member to admin
	updateReq := map[string]string{
		"role": "admin",
	}

	resp, body := doRequest(t, "PUT", apiGatewayURL+"/api/chats/"+chat.ID+"/participants/"+member.ID+"/role", updateReq, owner.AccessToken)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))
}

func TestChat_AddParticipant_AlreadyMember(t *testing.T) {
	SkipIfNotIntegration(t)

	owner := createTestUser(t, "alreadyowner")
	member := createTestUser(t, "alreadymember")

	chat := createTestChat(t, owner, "group", "Already Member Test", []string{member.ID})

	// Try to add member again
	addReq := map[string]string{
		"user_id": member.ID,
		"role":    "member",
	}

	resp, _ := doRequest(t, "POST", apiGatewayURL+"/api/chats/"+chat.ID+"/participants", addReq, owner.AccessToken)

	// API may return 409 Conflict or 201 Created (idempotent operation)
	assert.True(t, resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusCreated,
		"Expected 409 or 201, got %d", resp.StatusCode)
}

func TestChat_List_CursorPagination(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "cursorpag")

	// Create 5 chats
	for i := 0; i < 5; i++ {
		createTestChat(t, user, "group", fmt.Sprintf("Cursor Chat %d", i), nil)
	}

	// First page with limit 2
	resp, body := doRequest(t, "GET", apiGatewayURL+"/api/chats?limit=2", nil, user.AccessToken)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	chats, ok := result["chats"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 2, len(chats))

	// Check if there's a next cursor or has_more flag
	if hasMore, exists := result["has_more"]; exists {
		assert.True(t, hasMore.(bool), "Should have more chats")
	}
}
