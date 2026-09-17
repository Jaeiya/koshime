package views

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRatingModelParseRating(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantRating  int
		wantErr     bool
		wantErrText string
	}{
		// Missing value
		{
			name:        "empty",
			input:       "",
			wantErr:     true,
			wantErrText: "missing value",
		},
		{
			name:        "whitespace only",
			input:       "   ",
			wantErr:     true,
			wantErrText: "missing value",
		},
		{
			name:        "trimmed empty",
			input:       "  \t\n ",
			wantErr:     true,
			wantErrText: "missing value",
		},

		// Integer ratings
		{
			name:       "int 1",
			input:      "1",
			wantRating: 2,
		},
		{
			name:       "int 5",
			input:      "5",
			wantRating: 10,
		},
		{
			name:       "int 10",
			input:      "10",
			wantRating: 20,
		},
		{
			name:       "int with surrounding whitespace",
			input:      " 7 ",
			wantRating: 14,
		},
		{
			name:        "int below min range should err",
			input:       "0",
			wantErr:     true,
			wantErrText: "rating exceeds expected range",
		},
		{
			name:        "negative int should err",
			input:       "-1",
			wantErr:     true,
			wantErrText: "rating exceeds expected range",
		},
		{
			name:        "int above max range should err",
			input:       "11",
			wantErr:     true,
			wantErrText: "rating exceeds expected range",
		},
		{
			name:        "non-numeric int should err",
			input:       "abc",
			wantErr:     true,
			wantErrText: "invalid rating",
		},
		{
			name:        "int with trailing text",
			input:       "5abc",
			wantErr:     true,
			wantErrText: "invalid rating",
		},

		// Float ratings
		{
			name:       "one point five",
			input:      "1.5",
			wantRating: 3,
		},
		{
			name:       "five point five",
			input:      "5.5",
			wantRating: 11,
		},
		{
			name:       "float with zero fraction",
			input:      "5.0",
			wantRating: 10,
		},
		{
			name:        "ten point five exceeds range",
			input:       "10.5",
			wantErr:     true,
			wantErrText: "rating exceeds expected range",
		},
		{
			name:        "five point two is invalid",
			input:       "5.2",
			wantErr:     true,
			wantErrText: "invalid fraction; increment by 0.5",
		},
		{
			name:        "five point seven is invalid",
			input:       "5.7",
			wantErr:     true,
			wantErrText: "invalid fraction; increment by 0.5",
		},
		{
			name:        "fraction only",
			input:       ".5",
			wantErr:     true,
			wantErrText: "invalid float value",
		},
		{
			name:        "missing fractional part",
			input:       "5.",
			wantErr:     true,
			wantErrText: "invalid float value",
		},
		{
			name:        "multiple decimal points",
			input:       "5.5.5",
			wantErr:     true,
			wantErrText: "invalid float value",
		},
		{
			name:        "non-numeric float should err",
			input:       "x.5",
			wantErr:     true,
			wantErrText: "invalid float value",
		},
		{
			name:        "non-numeric fraction should err",
			input:       "5.x",
			wantErr:     true,
			wantErrText: "invalid float value",
		},
		{
			name:        "float outside upper range",
			input:       "11.5",
			wantErr:     true,
			wantErrText: "rating exceeds expected range",
		},
		{
			name:        "float outside lower range",
			input:       "0.5",
			wantErr:     true,
			wantErrText: "rating exceeds expected range",
		},
		{
			name:        "negative float",
			input:       "-1.5",
			wantErr:     true,
			wantErrText: "rating exceeds expected range",
		},
		{
			name:       "float with surrounding whitespace",
			input:      " 8.5 ",
			wantRating: 17,
		},
	}

	m := RatingModel{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := m.parseRating(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErrText)
				assert.Equal(t, -1, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantRating, got)
		})
	}
}
