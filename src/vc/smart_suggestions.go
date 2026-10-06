package vc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/dl"
	"ashokshau/tgmusic/src/utils"
	td "github.com/AshokShau/gotdbot"
)

type SmartSuggestion struct {
	ID      string
	Title   string
	Artist  string
	URL     string
	VideoID string
}

type smartSuggestionSet struct {
	SourceID string
	Items    map[string]SmartSuggestion
	Consumed bool
}

var smartSuggestions = struct {
	sync.Mutex
	sets map[int64]*smartSuggestionSet
}{
	sets: make(map[int64]*smartSuggestionSet),
}

func ClearSmartSuggestions(chatID int64) {
	smartSuggestions.Lock()
	defer smartSuggestions.Unlock()
	delete(smartSuggestions.sets, chatID)
}

func buildSmartSuggestions(sourceID string, tracks []utils.MusicTrack) []SmartSuggestion {
	seen := make(map[string]bool)
	out := make([]SmartSuggestion, 0, 5)

	for _, t := range tracks {
		if t.Id == "" || t.Id == sourceID || t.Url == "" || t.Title == "" || seen[t.Id] {
			continue
		}

		seen[t.Id] = true

		idBytes := make([]byte, 8)
		if _, err := rand.Read(idBytes); err != nil {
			continue
		}

		out = append(out, SmartSuggestion{
			ID:      hex.EncodeToString(idBytes),
			Title:   t.Title,
			Artist:  t.Channel,
			URL:     t.Url,
			VideoID: t.Id,
		})

		if len(out) == 5 {
			break
		}
	}

	return out
}

func GenerateSmartSuggestions(
	ctx context.Context,
	chatID int64,
	source *utils.CachedTrack,
) ([]SmartSuggestion, error) {
	if source == nil || source.TrackID == "" {
		return nil, errors.New("missing source track")
	}

	tracks, err := dl.GetYouTubeMixPlaylist(ctx, "RD"+source.TrackID)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	out := make([]SmartSuggestion, 0, 5)

	for _, t := range tracks.Results {
		if t.Id == "" || t.Id == source.TrackID || seen[t.Id] {
			continue
		}
		seen[t.Id] = true

		idBytes := make([]byte, 8)
		if _, err := rand.Read(idBytes); err != nil {
			continue
		}

		out = append(out, SmartSuggestion{
			ID:      hex.EncodeToString(idBytes),
			Title:   t.Title,
			Artist:  t.Channel,
			URL:     t.Url,
			VideoID: t.Id,
		})

		if len(out) >= 5 {
			break
		}
	}

	if len(out) == 0 {
		return nil, nil
	}

	items := make(map[string]SmartSuggestion, len(out))
	for _, s := range out {
		items[s.ID] = s
	}

	smartSuggestions.Lock()
	smartSuggestions.sets[chatID] = &smartSuggestionSet{
		SourceID: source.TrackID,
		Items:    items,
	}
	smartSuggestions.Unlock()

	return out, nil
}

func GetSmartSuggestions(chatID int64) []SmartSuggestion {
	smartSuggestions.Lock()
	defer smartSuggestions.Unlock()

	set := smartSuggestions.sets[chatID]
	if set == nil || set.Consumed {
		return nil
	}

	out := make([]SmartSuggestion, 0, len(set.Items))
	for _, item := range set.Items {
		out = append(out, item)
	}
	return out
}

func (c *TelegramCalls) PlaySmartSuggestion(
	bot *td.Client,
	chatID int64,
	suggestionID string,
) error {
	smartSuggestions.Lock()

	set := smartSuggestions.sets[chatID]
	if set == nil || set.Consumed {
		smartSuggestions.Unlock()
		return errors.New("suggestion expired")
	}

	suggestion, ok := set.Items[suggestionID]
	if !ok {
		smartSuggestions.Unlock()
		return errors.New("suggestion not found")
	}

	set.Consumed = true
	delete(smartSuggestions.sets, chatID)
	smartSuggestions.Unlock()

	track := &utils.CachedTrack{
		URL:      suggestion.URL,
		Name:     suggestion.Title,
		User:     "Smart Suggestions",
		TrackID:  suggestion.VideoID,
		Channel:  suggestion.Artist,
		Platform: utils.YouTube,
		IsVideo:  false,
	}

	cache.ChatCache.AddSong(chatID, track)

	return c.playSong(bot, chatID, track)
}
