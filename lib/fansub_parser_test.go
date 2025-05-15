package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type parseMock struct {
	should   string
	actual   string
	expected FansubInfo
	err      error
}

func TestFansubParser(t *testing.T) {
	mocks := []parseMock{
		{
			should: "support ellipses-separated file names with fansub group",
			// Assume fansub groups are one word when ellipses are involved
			actual: "some.kind.of.title.with.ellipses-FansubGroup",
			expected: FansubInfo{
				Fansub: "FansubGroup",
				Title:  "some kind of title with ellipses",
			},
		},
		{
			should: "support space-separated file names with fansub group",
			// Fansub groups can be one or more words when spaces are involved
			actual: "[Fansub Group] some kind of title with spaces",
			expected: FansubInfo{
				Fansub: "Fansub Group",
				Title:  "some kind of title with spaces",
			},
		},
		{
			should: "support underscore-separated file names with fansub group",
			// Fansub groups will be one word or use underscores when underscores are involved
			actual: "[Fansub_Group]_some_kind_of_title_with_underscores",
			expected: FansubInfo{
				Fansub: "Fansub_Group",
				Title:  "some kind of title with underscores",
			},
		},
		{
			should: "support inconsistent separators mixed with space separators",
			actual: "some.kind.of.title.1080p.h.265-FansubGroup (some other stuff here)",
			expected: FansubInfo{
				Fansub:   "FansubGroup",
				Title:    "some kind of title",
				Encoding: "1080p HEVC",
			},
		},
		{
			should: "support S##E## episode format",
			actual: "some kind of title S02E10",
			expected: FansubInfo{
				Title:   "some kind of title",
				Season:  "02",
				Episode: "10",
			},
		},
		{
			should: "support 'Season # - #' episode format",
			actual: "some kind of title Season 5 - 20",
			expected: FansubInfo{
				Title:   "some kind of title",
				Season:  "5",
				Episode: "20",
			},
		},
		{
			should: "support 'S# - #' episode format",
			actual: "some kind of title S10 - 09",
			expected: FansubInfo{
				Title:   "some kind of title",
				Season:  "10",
				Episode: "09",
			},
		},
		{
			should: "support '- #' episode format",
			actual: "some kind of title - 07",
			expected: FansubInfo{
				Title:   "some kind of title",
				Episode: "07",
			},
		},
		{
			should: "support '1st Season - #' episode format",
			actual: "some kind of title 1st Season - 1",
			expected: FansubInfo{
				Title:   "some kind of title",
				Season:  "1",
				Episode: "1",
			},
		},
		{
			should: "support '2nd Season - #' episode format",
			actual: "some kind of title 2nd Season - 5",
			expected: FansubInfo{
				Title:   "some kind of title",
				Season:  "2",
				Episode: "5",
			},
		},
		{
			should: "support '3rd Season - #' episode format",
			actual: "some kind of title 23rd Season - 05",
			expected: FansubInfo{
				Title:   "some kind of title",
				Season:  "23",
				Episode: "05",
			},
		},
		{
			should: "support '#th Season - #' episode format",
			actual: "some kind of title 99th Season - 100",
			expected: FansubInfo{
				Title:   "some kind of title",
				Season:  "99",
				Episode: "100",
			},
		},
		{
			should: "support '<title> # [<meta-data>' episode format",
			actual: "some kind of title 27 [1080p]",
			expected: FansubInfo{
				Title:    "some kind of title",
				Episode:  "27",
				Encoding: "1080p",
			},
		},
		{
			should: "support 'S#' season format",
			actual: "some kind of title S01",
			expected: FansubInfo{
				Title:  "some kind of title",
				Season: "01",
			},
		},
		{
			should: "ignore episode formats beyond meta-data",
			actual: "some kind of title S01 [1080p] S05E45",
			expected: FansubInfo{
				Title:    "some kind of title",
				Season:   "01",
				Encoding: "1080p",
			},
		},
	}

	t.Parallel()

	for _, mock := range mocks {
		t.Run("should "+mock.should, func(t *testing.T) {
			a := assert.New(t)
			var fs Fansub
			info, err := fs.Parse(mock.actual)
			a.NoError(err)
			a.Equal(mock.expected, info)
		})
	}
}
