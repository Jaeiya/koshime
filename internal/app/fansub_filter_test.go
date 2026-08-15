package app

import (
	"testing"

	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScore(t *testing.T) {
	ff := FansubFilter{}
	t.Parallel()

	tests := []struct {
		desc      string
		title     string
		anime     kitsu.Anime
		wantScore int
	}{
		{
			desc:      "handle empty titles",
			anime:     kitsu.Anime{JPN_Title: "some title"},
			wantScore: 0,
		},
		{
			desc:      "handle white space titles",
			title:     "   \t  ",
			anime:     kitsu.Anime{JPN_Title: "some title"},
			wantScore: 0,
		},
		{
			desc:      "match jpn title exactly",
			title:     "Shingeki no Kyojin",
			anime:     kitsu.Anime{JPN_Title: "Shingeki no Kyojin"},
			wantScore: 100,
		},
		{
			desc:  "match eng title exactly",
			title: "Attack on Titan",
			anime: kitsu.Anime{
				JPN_Title: "Shingeki no Kyoujin",
				ENG_Title: "Attack on Titan",
			},
			wantScore: 100,
		},
		{
			desc:      "match alt title exactly",
			title:     "AoT",
			anime:     kitsu.Anime{AltTitles: []string{"SnK", "AoT"}},
			wantScore: 100,
		},
		{
			desc:      "match as substr of title",
			title:     "Hero Academia",
			anime:     kitsu.Anime{JPN_Title: "My Hero Academia"},
			wantScore: 66,
		},
		{
			desc:      "match all possible ordered words",
			title:     "My Academia",
			anime:     kitsu.Anime{ENG_Title: "My Hero Academia"},
			wantScore: 66,
		},
		{
			desc:      "match all possible unordered words",
			title:     "Academia Hero My",
			anime:     kitsu.Anime{ENG_Title: "My Hero Academia"},
			wantScore: 100,
		},
	}

	for _, tt := range tests {
		t.Run("should "+tt.desc, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantScore, ff.score(tt.title, tt.anime))
		})
	}
}

func TestFilterNamesByAnime(t *testing.T) {
	t.Parallel()
	fs := utils.FileSys{}

	targetAnime := kitsu.Anime{
		ENG_Title: "My Hero Academia",
		JPN_Title: "Boku no Hero Academia",
	}

	tests := []struct {
		name      string
		anime     kitsu.Anime
		stream    utils.FilenameIterator
		want      []string
		errSubstr string
	}{
		{
			name:   "return nil with no files found",
			anime:  targetAnime,
			stream: fs.GenFilenameStream(),
			want:   nil,
		},
		{
			name:   "skip unsupported file extensions",
			anime:  targetAnime,
			stream: fs.GenFilenameStream("file.txt", "file.png", "file.exe"),
			want:   nil,
		},
		{
			name:  "return only file names matching the highest score",
			anime: targetAnime,
			stream: fs.GenFilenameStream(
				"[SubsPlease] Boku no Hero Academia - 01 [1080p].mkv", // High score (100)
				"[SubsPlease] Hero Academia - 02 [720p].mkv",          // Low score (50)
				"[SubsPlease] Naruto - 01 [1080p].mkv",                // No score (0)
			),
			want: []string{
				"[SubsPlease] Boku no Hero Academia - 01 [1080p].mkv",
			},
		},
		{
			name:  "match multiple files sharing the same top score",
			anime: targetAnime,
			stream: fs.GenFilenameStream(
				"[SubsPlease] Boku no Hero Academia - 01 [1080p].mkv", // Score 100
				"[Erai-raws] Boku no Hero Academia - 02 [1080p].mkv",  // Score 100
				"[SubsPlease] Other Show - 01 [1080p].mkv",            // Score 0
			),
			want: []string{
				"[SubsPlease] Boku no Hero Academia - 01 [1080p].mkv",
				"[Erai-raws] Boku no Hero Academia - 02 [1080p].mkv",
			},
		},
	}

	for _, tt := range tests {
		t.Run("should "+tt.name, func(t *testing.T) {
			t.Parallel()
			ff := FansubFilter{}
			got, err := ff.FilterFilenamesByAnime(tt.anime, tt.stream)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
