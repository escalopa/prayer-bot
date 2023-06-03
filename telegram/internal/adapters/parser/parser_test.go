package parser

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/escalopa/gopray/telegram/internal/adapters/memory"
	"github.com/stretchr/testify/require"
)

func TestParser_ParseSchedule(t *testing.T) {
	ctx := context.Background()
	loc, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)

	days := "День,ФАЖР,ВОСХОД,ЗУХР,АСР,МАГРИБ,ИША"
	pr := memory.NewPrayerRepository()
	tests := []struct {
		name    string
		data    []string
		wantErr bool
	}{
		{
			name: "Test 1",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13,11:47,13:34,15:21,17:18\n",
				"2/1,5:53,8:13,11:47,13:34,15:21,17:18\n",
			},
			wantErr: false,
		},
		{
			name: "Test 2",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13,11:47,13:34,15:21,1718\n", // wrong time
			},
			wantErr: true,
		},
		{
			name: "Test 3",
			data: []string{
				days + "\n",
				"1/,5:53,8:13,11:47,13:34,15:21,17:18\n", // wrong month
			},
			wantErr: true,
		},
		{
			name: "Test 4",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13,11:4713:34,15:21,17:18\n", // wrong separator
			},
			wantErr: true,
		},
		{
			name: "Test 5",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13,11:47,13:34,15:21,17:18\n",
				"2/1,5:53,8:13,11:47,13:34,15:21,17:70\n", // wrong time, minutes > 59
			},
			wantErr: true,
		},
		{
			name: "Test 6",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13,11:47,13:34,15:21,17:-1\n", // wrong time, minutes < 0
			},
			wantErr: true,
		},
		{
			name: "Test 7",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13,11:47,13:34,15:21,24:18\n", // wrong time, hours > 23
			},
			wantErr: true,
		},
		{
			name: "Test 8",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13,11:47,13:34,15:21,-1:18\n", // wrong time, hours < 0
			},
			wantErr: true,
		},
		{
			name: "Test 9",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13,11:47,13:34,15:21,17;18\n", // wrong separator in time, used ; instead of :
			},
			wantErr: true,
		},
		{
			name: "Test 10",
			data: []string{
				days + "\n",
				"1/15:53,8:13,11:47,13:34,15:21,17:18\n", // removed one comma
			},
			wantErr: true,
		},
		{
			name: "Test 11",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13,11:47,13:34,15:21,1s:18\n", // wrong time, s instead of a number in hour
			},
			wantErr: true,
		},
		{
			name: "Test 12",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13,11:47,13:34,15:21,17:1s\n", // wrong time, s instead of a number in minutes
			},
			wantErr: true,
		},
		{
			name: "Test 13",
			data: []string{
				days + "\n",
				"s/1,5:53,8:13,11:47,13:34,15:21,17:18\n", // wrong day, s instead of a number
			},
			wantErr: true,
		},
		{
			name: "Test 14",
			data: []string{
				days + "\n",
				"1/s,5:53,8:13,11:47,13:34,15:21,17:18\n", // wrong month, s instead of a number
			},
			wantErr: true,
		},
		{
			name: "Test 15",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13'10:47,13:34,15:21,17:18\n", // wrong separator in time, used ' instead of :
			},
			wantErr: true,
		},
		{
			name: "Test 16",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13,11:47,13:34,15:21,17:18,1:2:3\n", // too many columns
			},
			wantErr: true,
		},
		{
			name: "Test 17",
			data: []string{
				days + "\n",
				"1/1,50:53,8:13,11:47,13:34,16:21,17:18\n", // wrong time for fajr, hours > 24
			},
			wantErr: true,
		},
		{
			name: "Test 18",
			data: []string{
				days + "\n",
				"1/1,5:53,80:13,11:47,13:34,15:21,17:18\n", // wrong time for sunrise, hours > 24
			},
			wantErr: true,
		},
		{
			name: "Test 19",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13,35:47,13:34,15:21,17:18\n", // wrong time for dhuhr, hours > 24
			},
			wantErr: true,
		},
		{
			name: "Test 20",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13,11:47,134:34,15:21,17:18\n", // wrong time for asr, hours > 24
			},
			wantErr: true,
		},
		{
			name: "Test 21",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13,11:47,13:34,155:21,17:18\n", // wrong time for maghrib, hours > 24
			},
			wantErr: true,
		},
		{
			name: "Test 22",
			data: []string{
				days + "\n",
				"1/1,5:53,8:13,11:47,13:34,15:21,177:18\n", // wrong time for isha, hours > 24
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			file, err := os.CreateTemp("", "test")
			require.NoError(t, err)
			defer func() {
				require.NoError(t, file.Close())
				require.NoError(t, os.Remove(file.Name()))
			}()
			// Write data to the file
			for _, line := range tt.data {
				_, err := file.WriteString(line)
				require.NoError(t, err)
			}
			// Create a parser with the path to the file
			p := NewPrayerParser(file.Name(), WithTimeLocation(loc), WithPrayerRepository(pr))
			// Parse the file
			err = p.ParseSchedule(ctx)
			require.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestParser_ConvertToTime(t *testing.T) {
	type input struct {
		day   int
		month int
		time  string
	}

	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Errorf("time.LoadLocation() error = %v, wantErr %v", err, false)
		return
	}

	p := NewPrayerParser("RANDOM_PATH", WithTimeLocation(loc))
	tests := []struct {
		name string
		args input
		want time.Time
	}{
		{
			name: "Test 1",
			args: input{day: 1, month: 1, time: "05:30"},
			want: time.Date(time.Now().Year(), 1, 1, 5, 30, 0, 0, loc),
		},
		{
			name: "Test 2",
			args: input{day: 4, month: 5, time: "15:35"},
			want: time.Date(time.Now().Year(), 5, 4, 15, 35, 0, 0, loc),
		},
		{
			name: "Test 3",
			args: input{day: 31, month: 12, time: "23:59"},
			want: time.Date(time.Now().Year(), 12, 31, 23, 59, 0, 0, loc),
		},
		{
			name: "Test 4",
			args: input{day: 1, month: 1, time: "00:00"},
			want: time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.convertToTime(tt.args.time, tt.args.day, tt.args.month)
			if err != nil {
				t.Errorf("convertToTime() error = %v, wantErr %v", err, false)
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("convertToTime() got = %v, want %v", got, tt.want)
			}
		})
	}
}
