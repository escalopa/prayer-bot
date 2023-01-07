package prayer

import (
	"reflect"
	"testing"
	"time"
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
	if err := p2.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(p1, p2) {
		t.Fatalf("expected %v, got %v", p1, p2)
	}
}
