/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package core

import (
	"ashokshau/tgmusic/config"
	"fmt"

	"github.com/AshokShau/gotdbot"
)

func cb(text, data string) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text: text,
		Type: &gotdbot.InlineKeyboardButtonTypeCallback{
			Data: []byte(data),
		},
	}
}

func cbCustom(text, data string, emojiID int64) gotdbot.InlineKeyboardButton {
	btn := cb(text, data)
	btn.IconCustomEmojiId = emojiID
	return btn
}

func urlCustom(text, link string, emojiID int64) gotdbot.InlineKeyboardButton {
	btn := url(text, link)
	btn.IconCustomEmojiId = emojiID
	return btn
}

// cbStyled is cb() with an explicit button Style (Bot API 10.3's colored
// bot buttons - default/primary/success/danger/link).
func cbStyled(text, data string, style gotdbot.ButtonStyle) gotdbot.InlineKeyboardButton {
	btn := cb(text, data)
	btn.Style = style
	return btn
}

func url(text, link string) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text: text,
		Type: &gotdbot.InlineKeyboardButtonTypeUrl{
			Url: link,
		},
	}
}

// urlPrimary is url() with the button's Style set to the dark-blue "primary"
// color introduced for InlineKeyboardButton, used to make our single most
// important call-to-action (Add to Group) stand out from the plain-styled
// buttons around it.
func urlPrimary(text, link string) gotdbot.InlineKeyboardButton {
	btn := url(text, link)
	btn.Style = gotdbot.ButtonStylePrimary{}
	return btn
}

var CloseBtn = cbStyled("Close", "vcplay_close", gotdbot.ButtonStyleDanger{})
var HomeBtn = cbStyled("Home", "help_back", gotdbot.ButtonStylePrimary{})
var HelpBtn = cbStyled("Help", "help_all", gotdbot.ButtonStylePrimary{})
var UserBtn = cbStyled("Users", "help_user", gotdbot.ButtonStylePrimary{})
var AdminBtn = cbStyled("Admins", "help_admin", gotdbot.ButtonStylePrimary{})
var OwnerBtn = cbStyled("Owner", "help_owner", gotdbot.ButtonStylePrimary{})
var DevsBtn = cbStyled("Devs", "help_devs", gotdbot.ButtonStylePrimary{})
var PlaylistBtn = cbStyled("Playlist", "help_playlist", gotdbot.ButtonStylePrimary{})
var AutoplayBtn = cbStyled("Autoplay", "help_autoplay", gotdbot.ButtonStylePrimary{})

func SupportKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {

	channelBtn := url("Updates", config.SupportChannel)
	groupBtn := url("Group", config.SupportGroup)

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{channelBtn, groupBtn},
			{CloseBtn},
		},
	}
}

func SupportBtn() *gotdbot.ReplyMarkupInlineKeyboard {
	channelBtn := url("Updates", config.SupportChannel)
	groupBtn := url("Group", config.SupportGroup)
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{channelBtn, groupBtn},
		},
	}
}

func HelpMenuKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{UserBtn, AdminBtn, OwnerBtn},
			{PlaylistBtn, DevsBtn, AutoplayBtn},
			{HomeBtn, CloseBtn},
		},
	}
}

func BackHelpMenuKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{HelpBtn, HomeBtn},
			{CloseBtn},
		},
	}
}

// CancelDownloadKeyboard returns a single red "✗ Cancel" button used on the
// downloading/searching progress message. Pressing it closes (deletes) the
// message via the vcplay_close callback.
func CancelDownloadKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	cancelBtn := cbStyled("✗ Cancel", "vcplay_close", gotdbot.ButtonStyleDanger{})
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{cancelBtn},
		},
	}
}

func ControlButtons(mode string) *gotdbot.ReplyMarkupInlineKeyboard {
	skipBtn := cbCustom("Skip", "play_skip", 5472235990955334730)
	skipBtn.Style = gotdbot.ButtonStylePrimary{}
	stopBtn := cbCustom("Stop", "play_stop", 5472201536727686043)
	stopBtn.Style = gotdbot.ButtonStylePrimary{}
	pauseBtn := cbCustom("Pause", "play_pause", 5474288573005964288)
	pauseBtn.Style = gotdbot.ButtonStylePrimary{}
	resumeBtn := cbCustom("Resume", "play_resume", 5467470746215292572)
	resumeBtn.Style = gotdbot.ButtonStylePrimary{}
	muteBtn := cbCustom("", "play_mute", 5354963498075960423)
	unmuteBtn := cbCustom("", "play_unmute", 5352938317916683303)
	closeBtn := cbCustom("Close", "vcplay_close", 5417876320761696693)
	closeBtn.Style = gotdbot.ButtonStyleDanger{}
	addToPlaylistBtn := urlPrimary("Add Me  +", "https://t.me/YoutubeTunesbot?startgroup=true")
	addToPlaylistBtn.Style = gotdbot.ButtonStyleSuccess{}

	switch mode {
	case "play":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, pauseBtn},
				{addToPlaylistBtn, closeBtn},
			},
		}
	case "queue":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{addToPlaylistBtn, closeBtn},
			},
		}
	case "pause":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, resumeBtn},
				{closeBtn},
			},
		}
	case "resume":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, pauseBtn},
				{closeBtn},
			},
		}
	case "mute":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, unmuteBtn},
				{closeBtn},
			},
		}
	case "unmute":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, muteBtn},
				{closeBtn},
			},
		}
	default:
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{closeBtn},
			},
		}
	}
}

func AddMeMarkup(username string) *gotdbot.ReplyMarkupInlineKeyboard {

	addMeBtn := urlPrimary(
		"➕ Add me to your group",
		fmt.Sprintf("https://t.me/%s?startgroup=true", username),
	)

	channelBtn := url("Updates", config.SupportChannel)
	groupBtn := url("Group", config.SupportGroup)

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{addMeBtn},
			{HelpBtn},
			{channelBtn, groupBtn},
		},
	}
}

// SetupGuideBtn opens the step-by-step setup guide via callback.
var SetupGuideBtn = cb("Setup Guide", "setup_guide")

// StartBackBtn returns to the main /start panel via callback.
var StartBackBtn = cb("Back", "setup_back")

// PrivateStartMarkup builds the keyboard shown for /start in a private chat.
// Mirrors: Add to Group, Help & Commands, Support Chat / Updates, Setup Guide.
func PrivateStartMarkup(username string) *gotdbot.ReplyMarkupInlineKeyboard {
	addToGroupBtn := urlPrimary("➕  Add to Group", fmt.Sprintf("https://t.me/%s?startgroup=true", username))
	helpBtn := cbCustom("Help", "help_all", 5258023599419171861)
	helpBtn.Style = gotdbot.ButtonStyleDanger{}
	supportBtn := urlCustom("Support Chat", config.SupportGroup, 5276089339967716971)
	supportBtn.Style = gotdbot.ButtonStyleSuccess{}
	updatesBtn := urlCustom("Updates", config.SupportChannel, 5307943162486994719)
	updatesBtn.Style = gotdbot.ButtonStyleSuccess{}
	ownerBtn := urlCustom("Owner", "https://t.me/Thegrime", 5208878706717636743)
	ownerBtn.Style = gotdbot.ButtonStyleDanger{}

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{addToGroupBtn},
			{helpBtn},
			{supportBtn, updatesBtn},
			{ownerBtn},
		},
	}
}

// GroupWelcomeMarkup builds the keyboard shown when the bot is added to a group.
// JoinVerifyMarkup builds the single Verify button used for join-request verification.
// The group/user identifiers are kept only in the deep-link URL; the visible
// button text remains simply "Verify".
func JoinVerifyMarkup(username string, chatID, userID int64) *gotdbot.ReplyMarkupInlineKeyboard {
	verifyURL := fmt.Sprintf("https://t.me/%s?start=verify_%d_%d", username, chatID, userID)
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{url("Verify", verifyURL)},
		},
	}
}

func GroupWelcomeMarkup() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{SetupGuideBtn},
			{url("Support Chat", config.SupportGroup), url("Updates", config.SupportChannel)},
		},
	}
}

// GuestReplyMarkup builds the keyboard for the personalized card sent back
// via AnswerGuestQuery when someone summons the bot with its @username (or
// a reply) in a chat it isn't a member of yet — Telegram's "Guest Bots"
// feature. Only URL buttons are used here (no callbacks): the bot has no
// ongoing presence in that chat until it's actually added, so a callback
// button on this card wouldn't have anything reliable to call back into.
func GuestReplyMarkup(username string) *gotdbot.ReplyMarkupInlineKeyboard {
	addMeBtn := urlPrimary("➕ Add Me to Your Group", fmt.Sprintf("https://t.me/%s?startgroup=true", username))

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{addMeBtn},
		},
	}
}

// GuideBackMarkup is shown on the setup guide screen: Add to Group (which
// the guide text explicitly tells the user to tap), then Back and Close.
func GuideBackMarkup(username string) *gotdbot.ReplyMarkupInlineKeyboard {
	addToGroupBtn := urlPrimary("➕ Add to Group", fmt.Sprintf("https://t.me/%s?startgroup=true", username))
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{addToGroupBtn},
			{StartBackBtn, CloseBtn},
		},
	}
}
