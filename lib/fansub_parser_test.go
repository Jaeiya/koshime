package lib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type parseMock struct {
	should   string
	actual   string
	expected FansubFileInfo
	err      error
}

func TestFansubParser(t *testing.T) {
	mocks := []parseMock{
		{
			should: "support ellipses-separated file names with fansub group",
			// Assume fansub groups are one word when ellipses are involved
			actual: "some.kind.of.title.with.ellipses-FansubGroup",
			expected: FansubFileInfo{
				Fansub: "FansubGroup",
				Title:  "some kind of title with ellipses",
			},
		},
		{
			should: "support ellipses-separated file names with fansub at beginning",
			// Assume fansub groups are one word when ellipses are involved
			actual: "some.kind.of.title.with.ellipses-FansubGroup",
			expected: FansubFileInfo{
				Fansub: "FansubGroup",
				Title:  "some kind of title with ellipses",
			},
		},
		{
			should: "support multi-word fansub group when file contains spaces",
			// Fansub groups can be one or more words when spaces are involved
			actual: "[the fansub group] some kind of title with spaces",
			expected: FansubFileInfo{
				Fansub: "the fansub group",
				Title:  "some kind of title with spaces",
			},
		},
		{
			should: "support underscore-separated file names with fansub group",
			// Fansub groups will be one word or use underscores when underscores are involved
			actual: "[Fansub_Group]_some_kind_of_title_with_underscores",
			expected: FansubFileInfo{
				Fansub: "Fansub_Group",
				Title:  "some kind of title with underscores",
			},
		},
		{
			should: "support inconsistent separators mixed with space separators",
			actual: "some.kind.of.title.1080p.h.265-FansubGroup (some other stuff here)",
			expected: FansubFileInfo{
				Fansub:   "FansubGroup",
				Title:    "some kind of title",
				Encoding: "1080p HEVC",
			},
		},
		{
			should: "support S##E## episode format",
			actual: "some kind of title S02E10",
			expected: FansubFileInfo{
				Title:   "some kind of title",
				Season:  "02",
				Episode: "10",
			},
		},
		{
			should: "support S##E## version 2 episode format",
			actual: "some kind of title S02E10v2",
			expected: FansubFileInfo{
				Title:   "some kind of title",
				Season:  "02",
				Episode: "10v2",
			},
		},
		{
			should: "support 'Season # - #' episode format",
			actual: "some kind of title Season 5 - 20",
			expected: FansubFileInfo{
				Title:   "some kind of title",
				Season:  "5",
				Episode: "20",
			},
		},
		{
			should: "support 'Season # - #' version 2 episode format",
			actual: "some kind of title Season 5 - 20v2",
			expected: FansubFileInfo{
				Title:   "some kind of title",
				Season:  "5",
				Episode: "20v2",
			},
		},
		{
			should: "support 'S# - #' episode format",
			actual: "some kind of title S10 - 09",
			expected: FansubFileInfo{
				Title:   "some kind of title",
				Season:  "10",
				Episode: "09",
			},
		},
		{
			should: "support 'S# - #' version 2 episode format",
			actual: "some kind of title S10 - 09v2",
			expected: FansubFileInfo{
				Title:   "some kind of title",
				Season:  "10",
				Episode: "09v2",
			},
		},
		{
			should: "support '- #' episode format",
			actual: "some kind of title - 07",
			expected: FansubFileInfo{
				Title:   "some kind of title",
				Episode: "07",
			},
		},
		{
			should: "support '- #' version 2 episode format",
			actual: "some kind of title - 07v2",
			expected: FansubFileInfo{
				Title:   "some kind of title",
				Episode: "07v2",
			},
		},
		{
			should: "support '1st Season - #' episode format",
			actual: "some kind of title 1st Season - 1",
			expected: FansubFileInfo{
				Title:   "some kind of title",
				Season:  "1",
				Episode: "1",
			},
		},
		{
			should: "support '2nd Season - #' episode format",
			actual: "some kind of title 2nd Season - 5",
			expected: FansubFileInfo{
				Title:   "some kind of title",
				Season:  "2",
				Episode: "5",
			},
		},
		{
			should: "support '3rd Season - #' episode format",
			actual: "some kind of title 23rd Season - 05",
			expected: FansubFileInfo{
				Title:   "some kind of title",
				Season:  "23",
				Episode: "05",
			},
		},
		{
			should: "support '#th Season - #' episode format",
			actual: "some kind of title 99th Season - 100",
			expected: FansubFileInfo{
				Title:   "some kind of title",
				Season:  "99",
				Episode: "100",
			},
		},
		{
			should: "support '#th Season - #' version 2 episode format",
			actual: "some kind of title 99th Season - 100v2",
			expected: FansubFileInfo{
				Title:   "some kind of title",
				Season:  "99",
				Episode: "100v2",
			},
		},
		{
			should: "support '<title> # [<meta-data>' version 2 episode format",
			actual: "some kind of title 27v2 [1080p]",
			expected: FansubFileInfo{
				Title:    "some kind of title",
				Episode:  "27v2",
				Encoding: "1080p",
			},
		},
		{
			should: "support 'S#' season format",
			actual: "some kind of title S01",
			expected: FansubFileInfo{
				Title:  "some kind of title",
				Season: "01",
			},
		},
		{
			should: "support EP# episode format",
			actual: "some kind of title EP02",
			expected: FansubFileInfo{
				Title:   "some kind of title",
				Episode: "02",
			},
		},
		{
			should: "support EP# version 2 episode format",
			actual: "some kind of title EP02v2",
			expected: FansubFileInfo{
				Title:   "some kind of title",
				Episode: "02v2",
			},
		},
		{
			should: "ignore episode formats beyond meta-data",
			actual: "some kind of title S01 [1080p] S05E45",
			expected: FansubFileInfo{
				Title:    "some kind of title",
				Season:   "01",
				Encoding: "1080p",
			},
		},
		{
			should:   "error if invalid separator format",
			actual:   "some-kind-of-title-S01-[1080p]-S05E45",
			expected: FansubFileInfo{},
			err:      fmt.Errorf("unsupported file name"),
		},
		{
			should: "not duplicate encodings",
			actual: "some kind of title 1080p 1080p",
			expected: FansubFileInfo{
				Title:    "some kind of title",
				Encoding: "1080p",
			},
		},
		{
			should:   "not detect batch before meta-data",
			actual:   "some kind of batch title",
			expected: FansubFileInfo{Title: "some kind of batch title"},
		},
		{
			should: "detect batch file",
			actual: "some kind of batch title 1080p batch",
			expected: FansubFileInfo{
				Title:    "some kind of batch title",
				Encoding: "1080p",
			},
			err: fmt.Errorf("batch files not supported"),
		},
		{
			should: "support mixed title & meta-data separators",
			actual: "[fansub] some title [1080p.hevc.aac]",
			expected: FansubFileInfo{
				Fansub:   "fansub",
				Title:    "some title",
				Encoding: "1080p HEVC AAC",
			},
		},
		{
			should: "error with unsupported mixed separator variation",
			actual: "too short [1080p.hevc.aac]",
			expected: FansubFileInfo{
				Title:    "too short",
				Encoding: "1080p HEVC AAC",
			},
			err: fmt.Errorf("unsupported separator format"),
		},
		{
			should: "support aac 2.0 edge case",
			actual: "it is some title [aac.2.0]",
			expected: FansubFileInfo{
				Title:    "it is some title",
				Encoding: "AAC 2.0",
			},
		},
		{
			should: "support truehd 7.1 edge case",
			actual: "it is some title [truehd.7.1]",
			expected: FansubFileInfo{
				Title:    "it is some title",
				Encoding: "TrueHD 7.1",
			},
		},
		{
			should: "support separated 264 encoding",
			actual: "it is some title [1080p h 264]",
			expected: FansubFileInfo{
				Title:    "it is some title",
				Encoding: "1080p H.264",
			},
		},
		{
			should: "support separated 265 encoding",
			actual: "it is some title [1080p h 265]",
			expected: FansubFileInfo{
				Title:    "it is some title",
				Encoding: "1080p HEVC",
			},
		},
		{
			should: "support aac 2.0 separated format ",
			actual: "it is some title [1080p aac2 0]",
			expected: FansubFileInfo{
				Title:    "it is some title",
				Encoding: "1080p AAC 2.0",
			},
		},
		{
			should: "support ddp 5.1 separated format",
			actual: "it is some title [1080p ddp5 1]",
			expected: FansubFileInfo{
				Title:    "it is some title",
				Encoding: "1080p DolbyDigital+5.1",
			},
		},
		{
			should: "support ddp 2.0 separated format",
			actual: "it is some title [1080p ddp2 0]",
			expected: FansubFileInfo{
				Title:    "it is some title",
				Encoding: "1080p DolbyDigital+2.0",
			},
		},
		{
			should: "support full feature file names",
			actual: "[my fansub] it is some title S02E24v2 1080p nf web-dl aac2.0 h.264 (some extra info)",
			expected: FansubFileInfo{
				Fansub:   "my fansub",
				Title:    "it is some title",
				Encoding: "1080p AAC 2.0 H.264",
				Source:   "Netflix Web-Download",
				Season:   "02",
				Episode:  "24v2",
			},
		},
		{
			should: "support full ellipses separated file names",
			actual: "Re.ZERO.Starting.Life.in.Another.World.S03E16.FiNAL.MULTi.1080p.WEB-DL.x264-T3KASHi",
			expected: FansubFileInfo{
				Fansub:   "T3KASHi",
				Title:    "Re ZERO Starting Life in Another World",
				Encoding: "1080p H.264",
				Source:   "Web-Download",
				Season:   "03",
				Episode:  "16",
			},
		},
	}

	t.Parallel()

	for _, mock := range mocks {
		t.Run("should "+mock.should, func(t *testing.T) {
			a := assert.New(t)
			var fs FansubParser
			info, err := fs.Parse(mock.actual)
			if mock.err != nil {
				a.ErrorContains(err, mock.err.Error())
				return
			}
			a.NoError(err)
			a.Equal(mock.expected, info)
		})
	}
}
