package vc

import (
	"testing"

	"ashokshau/tgmusic/src/utils"
)

func TestBuildSmartSuggestionsFilteringAndMaximum(t *testing.T) {
	tracks := []utils.MusicTrack{
		{Id: "source", Title: "Source", Url: "https://youtube.com/source"},
		{Id: "", Title: "Invalid", Url: "https://youtube.com/invalid"},
		{Id: "a", Title: "A", Url: "https://youtube.com/a"},
		{Id: "a", Title: "Duplicate", Url: "https://youtube.com/a"},
		{Id: "b", Title: "B", Url: "https://youtube.com/b"},
		{Id: "c", Title: "C", Url: "https://youtube.com/c"},
		{Id: "d", Title: "D", Url: "https://youtube.com/d"},
		{Id: "e", Title: "E", Url: "https://youtube.com/e"},
		{Id: "f", Title: "F", Url: "https://youtube.com/f"},
	}

	got := buildSmartSuggestions("source", tracks)

	if len(got) != 5 {
		t.Fatalf("expected 5 suggestions, got %d", len(got))
	}

	seen := make(map[string]bool)
	for _, s := range got {
		if s.VideoID == "source" {
			t.Fatal("source track was suggested")
		}
		if s.VideoID == "" || s.URL == "" || s.Title == "" {
			t.Fatal("invalid suggestion returned")
		}
		if seen[s.VideoID] {
			t.Fatalf("duplicate suggestion returned: %s", s.VideoID)
		}
		seen[s.VideoID] = true
	}
}
