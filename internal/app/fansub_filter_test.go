package app

import (
	"testing"

	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFansubFilterByLibEntry(t *testing.T) {
	ff := FansubFilter{}
	fs := utils.FileSys{}
	t.Parallel()

	tests := []struct {
		name     string
		stream   utils.FilenameIterator
		actual   []kitsu.Anime
		expected []FilteredAnime
	}{
		{
			name:     "not error with empty input",
			stream:   fs.GenFilenameStream("some file name.mkv"),
			actual:   []kitsu.Anime{},
			expected: nil,
		},
		{
			name:   "return nil when no results found",
			stream: fs.GenFilenameStream("[group] some title.mkv"),
			actual: []kitsu.Anime{
				{
					JPN_Title: "the thing",
				},
			},
			expected: nil,
		},
		{
			name:   "not match lowest threshold anime title match",
			stream: fs.GenFilenameStream("[group] some title.mkv"),
			actual: []kitsu.Anime{
				{
					JPN_Title: "title",
				},
			},
			expected: nil,
		},
		{
			name:   "not match lowest threshold stream title match",
			stream: fs.GenFilenameStream("[group] title.mkv"),
			actual: []kitsu.Anime{
				{
					JPN_Title: "some title",
				},
			},
			expected: nil,
		},
		{
			name:   "do not match across title boundaries",
			stream: fs.GenFilenameStream("[group] this is not a match.mkv"),
			actual: []kitsu.Anime{
				{
					JPN_Title: "this",
					ENG_Title: "is not",
					AltTitles: []string{
						"a", "match",
					},
				},
			},
			expected: nil,
		},
		{
			name:   "match exactly",
			stream: fs.GenFilenameStream("[group] is a match.mkv"),
			actual: []kitsu.Anime{
				{
					JPN_Title: "is a match",
				},
			},
			expected: []FilteredAnime{
				{
					Anime: kitsu.Anime{JPN_Title: "is a match"},
					FileInfo: FansubFileInfo{
						Fansub:   "group",
						Title:    "is a match",
						Filename: "[group] is a match.mkv",
					},
					Score: 100,
				},
			},
		},
		{
			name:   "match exactly via fuzzy matching stream title",
			stream: fs.GenFilenameStream("[group] match is a.mkv"),
			actual: []kitsu.Anime{
				{
					JPN_Title: "is a match",
				},
			},
			expected: []FilteredAnime{
				{
					Anime: kitsu.Anime{JPN_Title: "is a match"},
					FileInfo: FansubFileInfo{
						Fansub:   "group",
						Title:    "match is a",
						Filename: "[group] match is a.mkv",
					},
					Score: 100,
				},
			},
		},
		{
			name:   "match exactly via fuzzy matching anime title",
			stream: fs.GenFilenameStream("[group] is a match.mkv"),
			actual: []kitsu.Anime{
				{
					JPN_Title: "match is a",
				},
			},
			expected: []FilteredAnime{
				{
					Anime: kitsu.Anime{JPN_Title: "match is a"},
					FileInfo: FansubFileInfo{
						Fansub:   "group",
						Title:    "is a match",
						Filename: "[group] is a match.mkv",
					},
					Score: 100,
				},
			},
		},
		{
			name:   "match partial anime title",
			stream: fs.GenFilenameStream("[group] a partial match.mkv"),
			actual: []kitsu.Anime{
				{
					JPN_Title: "partial match",
				},
			},
			expected: []FilteredAnime{
				{
					Anime: kitsu.Anime{JPN_Title: "partial match"},
					FileInfo: FansubFileInfo{
						Fansub:   "group",
						Title:    "a partial match",
						Filename: "[group] a partial match.mkv",
					},
					Score: 66,
				},
			},
		},
		{
			name:   "match partial stream title",
			stream: fs.GenFilenameStream("[group] partial match.mkv"),
			actual: []kitsu.Anime{
				{
					JPN_Title: "a partial match",
				},
			},
			expected: []FilteredAnime{
				{
					Anime: kitsu.Anime{JPN_Title: "a partial match"},
					FileInfo: FansubFileInfo{
						Fansub:   "group",
						Title:    "partial match",
						Filename: "[group] partial match.mkv",
					},
					Score: 66,
				},
			},
		},
		{
			name: "match on unique fansub titles only",
			stream: fs.GenFilenameStream(
				"[group] file name - 01.mkv",
				"[group] file name - 02.mkv",
				"[group] file name - 03.mkv",
				"[group] file name - 04.mkv",
				"[group] another file name - 01.mkv",
				"[group] another file name - 02.mkv",
				"[group] another file name - 03.mkv",
			),
			actual: []kitsu.Anime{
				{ID: "0", JPN_Title: "file name"},
				{ID: "1", JPN_Title: "another file name"},
			},
			expected: []FilteredAnime{
				{
					Anime: kitsu.Anime{ID: "0", JPN_Title: "file name"},
					FileInfo: FansubFileInfo{
						Fansub:   "group",
						Title:    "file name",
						Episode:  "01",
						Filename: "[group] file name - 01.mkv",
					},
					Score: 100,
				},
				{
					Anime: kitsu.Anime{ID: "1", JPN_Title: "another file name"},
					FileInfo: FansubFileInfo{
						Fansub:   "group",
						Title:    "another file name",
						Episode:  "01",
						Filename: "[group] another file name - 01.mkv",
					},
					Score: 100,
				},
			},
		},
		{
			name: "match only top scoring partial anime title match",
			stream: fs.GenFilenameStream(
				"[group] a b c d e f g h i j k l m n o p q r s t u v w x y z.mkv",
			),
			actual: []kitsu.Anime{
				{ // 61% match
					ID:        "0",
					JPN_Title: "a b c d e f g h i j k l m n o p",
				},
				{ // 53% match
					ID:        "1",
					JPN_Title: "a b c d e f g h i j k l m n",
				},
				{ // 84% match
					ID:        "2",
					JPN_Title: "a b c d e f g h i j k l m n o p q r s t u v",
				},
				{ // 73% match
					ID:        "3",
					JPN_Title: "a b c d e f g h i j k l m n o p q r s",
				},
			},
			expected: []FilteredAnime{
				{
					Anime: kitsu.Anime{
						ID:        "2",
						JPN_Title: "a b c d e f g h i j k l m n o p q r s t u v",
					},
					FileInfo: FansubFileInfo{
						Fansub:   "group",
						Title:    "a b c d e f g h i j k l m n o p q r s t u v w x y z",
						Filename: "[group] a b c d e f g h i j k l m n o p q r s t u v w x y z.mkv",
					},
					Score: 84,
				},
			},
		},
		{
			name: "match only top scoring partial stream title match",
			stream: fs.GenFilenameStream(
				"[group] a b c d e f g h i j k l m n o p.mkv",             // 61% match
				"[group] a b c d e f g h i j k l m n.mkv",                 // 53% match
				"[group] a b c d e f g h i j k l m n o p q r s t u v.mkv", // 84% match
				"[group] a b c d e f g h i j k l m n o p q r s.mkv",       // 73% match
			),
			actual: []kitsu.Anime{
				{JPN_Title: "a b c d e f g h i j k l m n o p q r s t u v w x y z"},
			},
			expected: []FilteredAnime{
				{
					Anime: kitsu.Anime{
						JPN_Title: "a b c d e f g h i j k l m n o p q r s t u v w x y z",
					},
					FileInfo: FansubFileInfo{
						Fansub:   "group",
						Title:    "a b c d e f g h i j k l m n o p q r s t u v",
						Filename: "[group] a b c d e f g h i j k l m n o p q r s t u v.mkv",
					},
					Score: 84,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run("should "+tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ff.FilterByLibEntry2(tt.stream, tt.actual)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

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
		{
			desc:      "match when anime word map is less than title length",
			title:     "My Hero Academia Extra Words",
			anime:     kitsu.Anime{ENG_Title: "My Hero Academia"},
			wantScore: 60,
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
