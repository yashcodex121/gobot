package handlers

import (
	"strconv"
	"strings"

	"ashokshau/tgmusic/src/core"

	td "github.com/AshokShau/gotdbot"
)

const joinVerifyText = "Click on the button below to verify yourself.\n\nYour request to join the group will be approved shortly."

// handleChatJoinRequest sends the requester a private Verify deep-link.
// The chat/user IDs are hidden in the URL behind the button; only "Verify"
// is visible to the user.
func handleChatJoinRequest(c *td.Client, u *td.UpdateNewChatJoinRequest) error {
	if u.Request == nil || u.UserChatId == 0 {
		return nil
	}

	userID := u.Request.UserId
	if userID == 0 {
		return nil
	}

	_, err := sendRich(
		c,
		u.UserChatId,
		joinVerifyText,
		core.JoinVerifyMarkup(c.Me.Usernames.EditableUsername, u.ChatId, userID),
	)
	if err != nil {
		c.Logger.Warn("failed to send join verification message", "chat_id", u.ChatId, "user_id", userID, "error", err)
		return nil
	}

	c.Logger.Info("sent join verification", "chat_id", u.ChatId, "user_id", userID)
	return nil
}

// handleVerifyStart processes /start verify_<chat_id>_<user_id>.
// The user ID embedded in the deep link must match the private chat that
// opened it, preventing one user from approving another user's request.
func handleVerifyStart(c *td.Client, m *td.Message) (bool, error) {
	arg := strings.TrimSpace(Args(m))
	if !strings.HasPrefix(arg, "verify_") || !m.IsPrivate() {
		return false, nil
	}

	parts := strings.Split(arg, "_")
	if len(parts) != 3 {
		_, err := replyRich(c, m, "This verification link is invalid or expired.", nil)
		return true, err
	}

	_, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		_, err = replyRich(c, m, "This verification link is invalid or expired.", nil)
		return true, err
	}

	userID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		_, err = replyRich(c, m, "This verification link is invalid or expired.", nil)
		return true, err
	}

	if m.ChatId != userID {
		_, err := replyRich(c, m, "This verification link belongs to another Telegram account.", nil)
		return true, err
	}

	// Verification only. Do NOT approve the group's join request.
	// Return false so startHandler continues into the normal private /start flow.
	return false, nil
}
