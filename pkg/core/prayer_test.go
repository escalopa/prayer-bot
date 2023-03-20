package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPrayerTimes_Marshal(t *testing.T) {
	p1 := New(
		1,
		1,
		time.Now().Add(time.Hour*1),
		time.Now().Add(time.Hour*2),
		time.Now().Add(time.Hour*3),
		time.Now().Add(time.Hour*4),
		time.Now().Add(time.Hour*5),
		time.Now().Add(time.Hour*6),
	)

	b, err := p1.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	p2 := PrayerTimes{}
	require.NoError(t, json.Unmarshal(b, &p2))

	// Compare p1 and p2
	require.Equal(t, p1.Day, p2.Day)
	require.Equal(t, p1.Month, p2.Month)
	require.WithinDurationf(t, p1.Fajr, p2.Fajr, time.Second, "Fajr")
	require.WithinDurationf(t, p1.Sunrise, p2.Sunrise, time.Second, "Sunrise")
	require.WithinDurationf(t, p1.Dhuhr, p2.Dhuhr, time.Second, "Dhuhr")
	require.WithinDurationf(t, p1.Asr, p2.Asr, time.Second, "Asr")
	require.WithinDurationf(t, p1.Maghrib, p2.Maghrib, time.Second, "Maghrib")
	require.WithinDurationf(t, p1.Isha, p2.Isha, time.Second, "Isha")
}
