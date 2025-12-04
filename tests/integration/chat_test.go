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
	assert.Equal(t, "group", result["type"])
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

	// Verify chat is deleted
	resp, _ = doRequest(t, "GET", apiGatewayURL+"/api/chats/"+chat.ID, nil, user.AccessToken)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
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

	// Verify participant cannot access chat
	resp, _ = doRequest(t, "GET", apiGatewayURL+"/api/chats/"+chat.ID, nil, participant.AccessToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestChat_Favorites(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "favorites")
	chat := createTestChat(t, user, "group", "Favorites Test Chat", nil)

	// Add to favorites
	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/chats/"+chat.ID+"/favorite", nil, user.AccessToken)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Response: %s", string(body))

	// Verify chat is in favorites
	resp, body = doRequest(t, "GET", apiGatewayURL+"/api/chats/"+chat.ID, nil, user.AccessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.True(t, result["is_favorite"].(bool))

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
	assert.True(t, result["is_archived"].(bool))

	// Unarchive chat
	resp, _ = doRequest(t, "DELETE", apiGatewayURL+"/api/chats/"+chat.ID+"/archive", nil, user.AccessToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
