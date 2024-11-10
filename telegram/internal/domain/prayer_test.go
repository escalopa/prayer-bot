package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPrayerTimesMarshal(t *testing.T) {
	now := time.Now()

	p1 := NewPrayerTime(
		now,
		now.Add(time.Hour*1),
		now.Add(time.Hour*2),
		now.Add(time.Hour*3),
		now.Add(time.Hour*4),
		now.Add(time.Hour*5),
		now.Add(time.Hour*6),
	)

	b, err := p1.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	p2 := PrayerTime{}
	require.NoError(t, json.Unmarshal(b, &p2))

	// Compare p1 and p2
	require.WithinDurationf(t, p1.Day, p2.Day, time.Second, "Day")
	require.WithinDurationf(t, p1.Fajr, p2.Fajr, time.Second, "Fajr")
	require.WithinDurationf(t, p1.Dohaa, p2.Dohaa, time.Second, "Dohaa")
	require.WithinDurationf(t, p1.Dhuhr, p2.Dhuhr, time.Second, "Dhuhr")
	require.WithinDurationf(t, p1.Asr, p2.Asr, time.Second, "Asr")
	require.WithinDurationf(t, p1.Maghrib, p2.Maghrib, time.Second, "Maghrib")
	require.WithinDurationf(t, p1.Isha, p2.Isha, time.Second, "Isha")
}
