package parser

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/escalopa/gopray/telegram/internal/adapters/memory"
	"github.com/stretchr/testify/require"
)

func TestParserParseSchedule(t *testing.T) {
	t.Parallel()

	const days = "День,ФАЖР,ВОСХОД,ЗУХР,АСР,МАГРИБ,ИША"

	tests := []struct {
		name  string
		data  []string
		check func(t *testing.T, err error)
	}{
		{
			name: "success",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13,11:47,13:34,15:21,17:18\n",
				"2/1/2023,5:53,8:13,11:47,13:34,15:21,17:18\n",
			},
			check: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "test_2",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13,11:47,13:34,15:21,1718\n", // wrong time
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_3",
			data: []string{
				days + "\n",
				"1//2023,5:53,8:13,11:47,13:34,15:21,17:18\n", // wrong month
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_4",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13,11:4713:34,15:21,17:18\n", // wrong separator
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_5",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13,11:47,13:34,15:21,17:18\n",
				"2/1/2023,5:53,8:13,11:47,13:34,15:21,17:70\n", // wrong time, minutes > 59
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_6",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13,11:47,13:34,15:21,17:-1\n", // wrong time, minutes < 0
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_7",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13,11:47,13:34,15:21,24:18\n", // wrong time, hours > 23
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_8",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13,11:47,13:34,15:21,-1:18\n", // wrong time, hours < 0
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_9",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13,11:47,13:34,15:21,17;18\n", // wrong separator in time, used ; instead of :
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_10",
			data: []string{
				days + "\n",
				"1/15/2023:53,8:13,11:47,13:34,15:21,17:18\n", // removed one comma
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_11",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13,11:47,13:34,15:21,1s:18\n", // wrong time, s instead of a number in hour
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_12",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13,11:47,13:34,15:21,17:1s\n", // wrong time, s instead of a number in minutes
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_13",
			data: []string{
				days + "\n",
				"s/1/2023,5:53,8:13,11:47,13:34,15:21,17:18\n", // wrong day, s instead of a number
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_14",
			data: []string{
				days + "\n",
				"1/s/2023,5:53,8:13,11:47,13:34,15:21,17:18\n", // wrong month, s instead of a number
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_15",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13'10:47,13:34,15:21,17:18\n", // wrong separator in time, used ' instead of :
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_16",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13,11:47,13:34,15:21,17:18,1:2:3\n", // too many columns
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_17",
			data: []string{
				days + "\n",
				"1/1/2023,50:53,8:13,11:47,13:34,16:21,17:18\n", // wrong time for fajr, hours > 24
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_18",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,80:13,11:47,13:34,15:21,17:18\n", // wrong time for sunrise, hours > 24
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_19",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13,35:47,13:34,15:21,17:18\n", // wrong time for dhuhr, hours > 24
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_20",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13,11:47,134:34,15:21,17:18\n", // wrong time for asr, hours > 24
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_21",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13,11:47,13:34,155:21,17:18\n", // wrong time for maghrib, hours > 24
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
		{
			name: "test_22",
			data: []string{
				days + "\n",
				"1/1/2023,5:53,8:13,11:47,13:34,15:21,177:18\n", // wrong time for isha, hours > 24
			},
			check: func(t *testing.T, err error) { require.Error(t, err) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			pr := memory.NewPrayerRepository()

			// Create a temporary file
			file, err := os.CreateTemp("", "test")
			require.NoError(t, err)
			defer func() {
				require.NoError(t, file.Close())
				require.NoError(t, os.Remove(file.Name()))
			}()

			// Write data to the file
			for _, line := range tt.data {
				_, err = file.WriteString(line)
				require.NoError(t, err)
			}

			// Create a parser with the path to the file
			parser := NewPrayerParser(file.Name(), pr, time.UTC)

			// Parse the file
			err = parser.LoadSchedule(ctx)
			tt.check(t, err)
		})
	}
}

func TestParserConvertToTime(t *testing.T) {
	t.Parallel()

	type input struct {
		day  time.Time
		time string
	}

	var year = time.Now().Year()

	tests := []struct {
		name string
		args input
		want time.Time
	}{
		{
			name: "test_1",
			args: input{
				day:  time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC),
				time: "05:30",
			},
			want: time.Date(year, 1, 1, 5, 30, 0, 0, time.UTC),
		},
		{
			name: "test_2",
			args: input{
				day:  time.Date(year, 5, 4, 0, 0, 0, 0, time.UTC),
				time: "15:35",
			},
			want: time.Date(year, 5, 4, 15, 35, 0, 0, time.UTC),
		},
		{
			name: "test_3",
			args: input{
				day:  time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC),
				time: "23:59",
			},
			want: time.Date(year, 12, 31, 23, 59, 0, 0, time.UTC),
		},
		{
			name: "test_4",
			args: input{
				day:  time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC),
				time: "00:00",
			},
			want: time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parser := NewPrayerParser("", nil, time.UTC)

			got, err := parser.convertToTime(tt.args.time, tt.args.day)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
