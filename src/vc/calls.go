/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package vc

/*
#cgo linux LDFLAGS: -L . -lntgcalls -lm -lz
#cgo darwin LDFLAGS: -L . -lntgcalls -lc++ -lz -lbz2 -liconv -framework AVFoundation -framework AudioToolbox -framework CoreAudio -framework QuartzCore -framework CoreMedia -framework VideoToolbox -framework AppKit -framework Metal -framework MetalKit -framework OpenGL -framework IOSurface -framework ScreenCaptureKit

// Currently is supported only dynamically linked library on Windows due to
// https://github.com/golang/go/issues/63903
#cgo windows LDFLAGS: -L. -lntgcalls
#include "ntgcalls/ntgcalls.h"
#include "glibc_compatibility.h"
*/
import "C"

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/utils"
	"ashokshau/tgmusic/src/vc/ntgcalls"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"math/big"
	"os"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

// getClientIndex selects an assistant client index (0-based) for a given chat.
func (c *TelegramCalls) getClientIndex(chatID int64) (int, error) {
	c.mu.RLock()
	totalClients := len(c.assistants)
	c.mu.RUnlock()

	if totalClients == 0 {
		return -1, fmt.Errorf("no clients are available")
	}

	assignedIndex, err := db.Instance.GetAssistant(chatID)
	if err != nil {
		slog.Info("[TelegramCalls] DB.GetAssistant error", "error", err)
		assignedIndex = -1
	}

	if assignedIndex >= 0 && assignedIndex < totalClients {
		return assignedIndex, nil
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(totalClients)))
	if err != nil {
		slog.Info("[TelegramCalls] Could not generate a random number", "error", err)
		newClientIndex := 0
		if assignedIndex == -1 && chatID != 0 {
			if _, err := db.Instance.AssignAssistant(chatID, newClientIndex); err != nil {
				logger.Info("[TelegramCalls] DB.AssignAssistant error", "error", err)
			}
		}
		return newClientIndex, nil
	}

	newClientIndex := int(n.Int64())
	if chatID != 0 {
		if _, err := db.Instance.AssignAssistant(chatID, newClientIndex); err != nil {
			logger.Info("[TelegramCalls] DB.AssignAssistant error", "error", err)
		}
	}

	return newClientIndex, nil
}

// GetGroupAssistant retrieves the assistant and its index for a given chat.
func (c *TelegramCalls) GetGroupAssistant(chatID int64) (*Assistant, int, error) {
	clientIndex, err := c.getClientIndex(chatID)
	if err != nil {
		return nil, -1, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	call, ok := c.assistants[clientIndex]
	if !ok {
		return nil, -1, fmt.Errorf("no ntgcalls instance was found for client index %d", clientIndex)
	}
	return call, clientIndex, nil
}

// dlProgressBar returns a 10-block progress bar at the given fill level (0-10).
func dlProgressBar(filled int) string {
	if filled < 0 {
		filled = 0
	}
	if filled > 10 {
		filled = 10
	}
	return strings.Repeat("▰", filled) + strings.Repeat("▱", 10-filled) +
		fmt.Sprintf(" %d%%", filled*10)
}

// sendDlPhotoMsg sends the downloading card for playSong: a photo (thumbnail)
// with a caption showing song info + progress bar + Cancel button.
// Falls back to a plain text message when no thumbnail is available or the
// photo send fails.
func sendDlPhotoMsg(bot *td.Client, chatID int64, song *utils.CachedTrack) (*td.Message, error) {
	escName := html.EscapeString(song.Name)
	escChannel := html.EscapeString(song.Channel)
	dur := utils.SecToMin(song.Duration)

	var caption string
	if escChannel != "" {
		caption = fmt.Sprintf(
			"<tg-emoji emoji-id=\"5346422717948716483\">⬇️</tg-emoji> <b>Downloading</b>\n\n"+
				"<tg-emoji emoji-id=\"5893297890117292323\">🔤</tg-emoji> <b>%s</b>\n"+
				"<tg-emoji emoji-id=\"5292226786229236118\">👤</tg-emoji> %s  •  <tg-emoji emoji-id=\"5893149782465057649\">⏱</tg-emoji> %s\n\n"+
				"%s",
			escName, escChannel, dur, dlProgressBar(3),
		)
	} else {
		caption = fmt.Sprintf(
			"<tg-emoji emoji-id=\"5346422717948716483\">⬇️</tg-emoji> <b>Downloading</b>\n\n"+
				"<tg-emoji emoji-id=\"5893297890117292323\">🔤</tg-emoji> <b>%s</b>\n"+
				"<tg-emoji emoji-id=\"5893149782465057649\">⏱</tg-emoji> %s\n\n"+
				"%s",
			escName, dur, dlProgressBar(3),
		)
	}

	cancelKeyboard := core.CancelDownloadKeyboard()

	if song.Thumbnail != "" {
		formattedCaption, err := bot.GetFormattedText(caption, nil, "HTML")
		if err == nil {
			photoMsg, err := bot.SendMessage(chatID, &td.InputMessagePhoto{
				Photo: &td.InputPhoto{
					Photo: td.InputFileRemote{Id: song.Thumbnail},
				},
				Caption: formattedCaption,
			}, &td.SendMessageOpts{ReplyMarkup: cancelKeyboard})
			if err == nil {
				return photoMsg, nil
			}
		}
	}

	// Fallback: plain text message.
	return bot.SendTextMessage(chatID, fmt.Sprintf("⬇️ Downloading: %s", song.Name), &td.SendTextMessageOpts{
		ReplyMarkup: cancelKeyboard,
	})
}

// editDlMsgToNowPlaying converts the downloading card to the Now Playing card.
// Handles both photo (edits caption) and text (edits text) messages.
func editDlMsgToNowPlaying(bot *td.Client, msg *td.Message, song *utils.CachedTrack) {
	escURL := html.EscapeString(song.URL)
	escName := html.EscapeString(song.Name)
	escUser := html.EscapeString(song.User)
	escChannel := html.EscapeString(song.Channel)

	var nowPlaying string
	if escChannel != "" {
		nowPlaying = fmt.Sprintf(
			"<tg-emoji emoji-id=\"5334665104677941170\">▶</tg-emoji> <u><b>Started streaming</b></u>\n\n"+
				"<tg-emoji emoji-id=\"5893297890117292323\">🔤</tg-emoji> <b>Title:</b> <a href='%s'>%s</a>\n"+
				"<tg-emoji emoji-id=\"5292226786229236118\">👤</tg-emoji> <b>Channel:</b> %s\n"+
				"<tg-emoji emoji-id=\"5893149782465057649\">⏱</tg-emoji> <b>Duration:</b> %s min\n"+
				"<tg-emoji emoji-id=\"5368324170671202286\">👋</tg-emoji> <b>Requested by:</b> %s",
			escURL, escName, escChannel, utils.SecToMin(song.Duration), escUser,
		)
	} else {
		nowPlaying = fmt.Sprintf(
			"<tg-emoji emoji-id=\"5334665104677941170\">▶</tg-emoji> <u><b>Started streaming</b></u>\n\n"+
				"<tg-emoji emoji-id=\"5893297890117292323\">🔤</tg-emoji> <b>Title:</b> <a href='%s'>%s</a>\n"+
				"<tg-emoji emoji-id=\"5893149782465057649\">⏱</tg-emoji> <b>Duration:</b> %s min\n"+
				"<tg-emoji emoji-id=\"5368324170671202286\">👋</tg-emoji> <b>Requested by:</b> %s",
			escURL, escName, utils.SecToMin(song.Duration), escUser,
		)
	}

	if _, isPhoto := msg.Content.(*td.MessagePhoto); isPhoto {
		formattedText, err := bot.GetFormattedText(nowPlaying, nil, "HTML")
		if err == nil {
			_, err = msg.EditCaption(bot, formattedText, &td.EditCaptionOpts{
				ReplyMarkup: core.ControlButtons("play"),
			})
			if err == nil {
				return
			}
		}
	}

	// Fallback: edit as plain HTML text.
	_, _ = msg.EditText(bot, nowPlaying, &td.EditTextMessageOpts{
		ReplyMarkup:           core.ControlButtons("play"),
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	})
}

// playSong downloads and plays a single song. It sends a downloading card
// (thumbnail photo + progress bar + Cancel button) while the track is being
// fetched, then converts it to the Now Playing card once streaming starts.
func (c *TelegramCalls) playSong(bot *td.Client, chatID int64, song *utils.CachedTrack) error {
	reply, err := sendDlPhotoMsg(bot, chatID, song)
	if err != nil {
		slog.Info("[playSong] Failed to send downloading message", "error", err)
		return err
	}

	if err = c.downloadAndPrepareSong(bot, song, reply); err != nil {
		return c.PlayNext(bot, chatID)
	}

	if err = c.PlayMedia(bot, chatID, song.FilePath, song.IsVideo, ""); err != nil {
		_, _ = bot.SendTextMessage(chatID, err.Error(), &td.SendTextMessageOpts{ParseMode: "HTML"})
		_ = bot.DeleteMessages(chatID, []int64{reply.Id}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	if song.Duration == 0 {
		song.Duration = utils.GetMediaDuration(song.FilePath)
	}
	c.schedulePrefetch(bot, chatID, song.Duration)

	editDlMsgToNowPlaying(bot, reply, song)
	return nil
}

// Stop halts media playback in a voice chat and clears the chat's cache.
func (c *TelegramCalls) Stop(chatId int64, banned bool) error {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return err
	}

	c.cancelPrefetch(chatId)
	c.clearPlayedOffset(chatId)
	c.clearAutoplayHistory(chatId)
	cache.ChatCache.SetAutoplay(chatId, false)
	cache.ChatCache.ClearChat(chatId)
	err = call.stopCall(chatId, banned)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}

		slog.Info("[Stop] Failed to stop the call", "error", err, "index", index)
		return fmt.Errorf("failed to stop call: %w", err)
	}
	return nil
}

// Pause temporarily stops media playback in a voice chat.
// It returns true if the operation was successful, and an error otherwise.
func (c *TelegramCalls) Pause(chatId int64) (bool, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}

	res, err := call.binding.Pause(chatId)
	if err != nil {
		slog.Warn("[Pause] Failed to pause the call", "error", err, "index", index)
		return res, fmt.Errorf("failed to pause: %w", err)
	}
	return res, err
}

// Resume continues a paused media playback in a voice chat.
func (c *TelegramCalls) Resume(chatId int64) (bool, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}

	res, err := call.binding.Resume(chatId)
	if err != nil {
		logger.Warn("Failed to resume the call", "error", err, "index", index)
		return res, fmt.Errorf("failed to resume: %w", err)
	}

	return res, err
}

// Mute silences the media playback in a voice chat.
func (c *TelegramCalls) Mute(chatId int64) (bool, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}

	res, err := call.binding.Mute(chatId)
	if err != nil {
		logger.Warn("Failed to mute the call", "error", err, "index", index)
		return res, fmt.Errorf("failed to mute: %w", err)
	}

	return res, err
}

// Unmute restores the audio of a muted media playback in a voice chat.
func (c *TelegramCalls) Unmute(chatId int64) (bool, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}

	res, err := call.binding.UnMute(chatId)
	if err != nil {
		logger.Warn("Failed to unmute the call", "error", err, "index", index)
		return res, fmt.Errorf("failed to unmute: %w", err)
	}

	return res, err
}

// PlayedTime retrieves the elapsed time of the current playback in a voice chat.
func (c *TelegramCalls) PlayedTime(chatId int64) (uint64, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return 0, err
	}

	_time, err := call.binding.Time(chatId, 0)
	if err != nil {
		logger.Warn("Failed to get played time", "error", err, "index", index)
		return 0, fmt.Errorf("failed to get played time: %w", err)
	}

	// ntgcalls' clock restarts with every new source (e.g. after a seek), so
	// add back the position that source started at.
	return _time + c.playedOffset(chatId), nil
}

// SeekStream jumps to a specific time in the current media stream.
func (c *TelegramCalls) SeekStream(bot *td.Client, chatID int64, filePath string, toSeek, duration int, isVideo bool) error {
	if toSeek < 0 || duration <= 0 {
		return errors.New("invalid seek position or duration. The position must be positive and the duration must be greater than 0")
	}

	isURL := urlRegex.MatchString(filePath)
	_, err := os.Stat(filePath)
	isFile := err == nil

	var ffmpegParams string
	if isURL || !isFile {
		ffmpegParams = fmt.Sprintf("-ss %d -i %s -to %d", toSeek, filePath, duration)
	} else {
		ffmpegParams = fmt.Sprintf("-ss %d -to %d", toSeek, duration)
	}

	// Remember where this source starts so PlayedTime stays accurate; roll
	// back if the seek fails and the old source keeps playing.
	prev := c.playedOffset(chatID)
	c.setPlayedOffset(chatID, uint64(toSeek))
	if err := c.PlayMedia(bot, chatID, filePath, isVideo, ffmpegParams); err != nil {
		c.setPlayedOffset(chatID, prev)
		return err
	}
	return nil
}

// RegisterHandlers sets up the event handlers for the voice call client.
func (c *TelegramCalls) RegisterHandlers(client *td.Client) {
	c.startAutoLeave(context.Background(), client)

	for _, call := range c.assistants {
		call.OnStreamEnd(func(chatID int64, streamType ntgcalls.StreamType, device ntgcalls.StreamDevice) {
			call.App.Logger.Infof("[OnStreamEnd] FIRED chat=%d stream_type=%v device=%v", chatID, streamType, device)

			if streamType == ntgcalls.VideoStream {
				call.App.Logger.Infof("[OnStreamEnd] Ignoring video stream chat=%d", chatID)
				return
			}

			call.App.Logger.Infof("[OnStreamEnd] Calling PlayNext chat=%d", chatID)

			if err := c.PlayNext(client, chatID); err != nil {
				call.App.Logger.Warnf("[OnStreamEnd] Failed to play the song chat=%d: %v", chatID, err)
			} else {
				call.App.Logger.Infof("[OnStreamEnd] PlayNext completed chat=%d", chatID)
			}
		})

		go func() {
			_, err := call.App.SendMessage(client.Me.Usernames.EditableUsername, "/start")
			if err != nil {
				call.App.Logger.Warnf("failed to start bot: %v", err)
			}

			_, err = call.App.SendMessage(config.LoggerId, "Userbot started.")
			if err != nil {
				call.App.Logger.Warnf("Failed to send message: %v", err)
			}
		}()
	}
}
