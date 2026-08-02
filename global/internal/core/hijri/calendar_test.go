package hijri

import (
	"testing"
	"time"
)

func TestFromGregorianUsesUmmAlQura(t *testing.T) {
	date, err := FromGregorian(time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC), 0)
	if err != nil {
		t.Fatal(err)
	}
	if date != (Date{Day: 6, Month: 5, Year: 1441}) {
		t.Fatalf("unexpected Umm al-Qura date: %+v", date)
	}
}

func TestFromGregorianAppliesRegionalAdjustment(t *testing.T) {
	base := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	date, err := FromGregorian(base, -1)
	if err != nil {
		t.Fatal(err)
	}
	if date != (Date{Day: 5, Month: 5, Year: 1441}) {
		t.Fatalf("unexpected adjusted date: %+v", date)
	}
}

func TestFromGregorianRejectsUnsafeAdjustment(t *testing.T) {
	if _, err := FromGregorian(time.Now(), 3); err == nil {
		t.Fatal("expected adjustment validation error")
	}
}
