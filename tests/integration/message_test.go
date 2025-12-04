package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessage_Send(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "msgsend")
	chat := createTestChat(t, user, "group", "Message Test Chat", nil)

	msgReq := map[string]string{
		"content": "Hello, World!",
	}

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/chats/"+chat.ID+"/messages", msgReq, user.AccessToken)

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Response: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.NotEmpty(t, result["id"])
	assert.Equal(t, "Hello, World!", result["content"])
	assert.Equal(t, chat.ID, result["chat_id"])
	assert.Equal(t, user.ID, result["sender_id"])
}

func TestMessage_List(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "msglist")
	chat := createTestChat(t, user, "group", "Message List Chat", nil)

	// Send a few messages
	for i := 0; i < 5; i++ {
		sendTestMessage(t, user, chat.ID, "Test message")
	}

	resp, body := doRequest(t, "GET", apiGatewayURL+"/api/chats/"+chat.ID+"/messages?limit=50", nil, user.AccessToken)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	messages, ok := result["messages"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(messages), 5)
}

func TestMessage_Update(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "msgupdate")
	chat := createTestChat(t, user, "group", "Update Message Chat", nil)
	msg := sendTestMessage(t, user, chat.ID, "Original content")

	updateReq := map[string]string{
		"content": "Updated content",
	}

	resp, body := doRequest(t, "PUT", apiGatewayURL+"/api/chats/messages/"+msg.ID, updateReq, user.AccessToken)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, "Updated content", result["content"])
	assert.True(t, result["is_edited"].(bool))
}

func TestMessage_Delete(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "msgdelete")
	chat := createTestChat(t, user, "group", "Delete Message Chat", nil)
	msg := sendTestMessage(t, user, chat.ID, "Message to delete")

	resp, _ := doRequest(t, "DELETE", apiGatewayURL+"/api/chats/messages/"+msg.ID, nil, user.AccessToken)

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestMessage_Reply(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "msgreply")
	chat := createTestChat(t, user, "group", "Reply Message Chat", nil)
	originalMsg := sendTestMessage(t, user, chat.ID, "Original message")

	replyReq := map[string]string{
		"content":     "Reply message",
		"reply_to_id": originalMsg.ID,
	}

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/chats/"+chat.ID+"/messages", replyReq, user.AccessToken)

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Response: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, originalMsg.ID, result["reply_to_id"])
}

func TestMessage_Reaction_Add(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "reaction")
	chat := createTestChat(t, user, "group", "Reaction Test Chat", nil)
	msg := sendTestMessage(t, user, chat.ID, "Message for reactions")

	reactionReq := map[string]string{
		"emoji": "👍",
	}

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/chats/messages/"+msg.ID+"/reactions", reactionReq, user.AccessToken)

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Response: %s", string(body))
}

func TestMessage_Reaction_Remove(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "rmreaction")
	chat := createTestChat(t, user, "group", "Remove Reaction Chat", nil)
	msg := sendTestMessage(t, user, chat.ID, "Message for removing reactions")

	// Add reaction first
	reactionReq := map[string]string{
		"emoji": "❤️",
	}
	doRequest(t, "POST", apiGatewayURL+"/api/chats/messages/"+msg.ID+"/reactions", reactionReq, user.AccessToken)

	// Remove reaction
	resp, _ := doRequest(t, "DELETE", apiGatewayURL+"/api/chats/messages/"+msg.ID+"/reactions/❤️", nil, user.AccessToken)

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestMessage_GuestCannotSend(t *testing.T) {
	SkipIfNotIntegration(t)

	// This test requires a user with guest role
	// For now, we skip it as it requires admin setup
	t.Skip("Requires guest user setup")
}

func TestMessage_Thread_Create(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "thread")
	chat := createTestChat(t, user, "group", "Thread Test Chat", nil)
	msg := sendTestMessage(t, user, chat.ID, "Message to thread")

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/chats/messages/"+msg.ID+"/thread", nil, user.AccessToken)

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Response: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.NotEmpty(t, result["id"])
}

func TestMessage_Typing(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "typing")
	chat := createTestChat(t, user, "group", "Typing Test Chat", nil)

	typingReq := map[string]bool{
		"is_typing": true,
	}

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/chats/"+chat.ID+"/typing", typingReq, user.AccessToken)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))
}
