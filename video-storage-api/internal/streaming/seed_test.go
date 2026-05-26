package streaming

import (
	"testing"
	"time"
)

func TestBuildSeedKey_StableForSameDay(t *testing.T) {
	morning := time.Date(2026, 5, 23, 8, 0, 0, 0, time.UTC)
	evening := time.Date(2026, 5, 23, 23, 59, 59, 0, time.UTC)

	a := BuildSeedKey("site-a", morning)
	b := BuildSeedKey("site-a", evening)
	if a != b {
		t.Fatalf("expected stable key within the day: %q vs %q", a, b)
	}
}

func TestBuildSeedKey_ChangesNextDay(t *testing.T) {
	today := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	tomorrow := today.Add(24 * time.Hour)

	if BuildSeedKey("site-a", today) == BuildSeedKey("site-a", tomorrow) {
		t.Fatal("expected key to change across days")
	}
}

func TestBuildSeedKey_PerSiteDifferent(t *testing.T) {
	at := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	if BuildSeedKey("site-a", at) == BuildSeedKey("site-b", at) {
		t.Fatal("expected different sites to produce different keys")
	}
}

func TestBuildSeedKey_EmptyMeansFresh(t *testing.T) {
	at := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	if got := BuildSeedKey("", at); got != "" {
		t.Fatalf("expected empty seed → empty key, got %q", got)
	}
}

func TestBuildSeedKey_NormalisesToUTC(t *testing.T) {
	// Same wall-clock instant, different zones — keys must match.
	loc, err := time.LoadLocation("Asia/Bangkok") // UTC+7
	if err != nil {
		t.Skip("tz database not available")
	}
	utc := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	bangkok := utc.In(loc)
	if BuildSeedKey("x", utc) != BuildSeedKey("x", bangkok) {
		t.Fatalf("expected UTC normalisation: %q vs %q",
			BuildSeedKey("x", utc), BuildSeedKey("x", bangkok))
	}
}

func TestClampLimit(t *testing.T) {
	cases := []struct {
		req, def, max int
		want          int
	}{
		{req: 0, def: 20, max: 50, want: 20},
		{req: -5, def: 20, max: 50, want: 20},
		{req: 10, def: 20, max: 50, want: 10},
		{req: 50, def: 20, max: 50, want: 50},
		{req: 999, def: 20, max: 50, want: 50},
	}
	for _, c := range cases {
		if got := ClampLimit(c.req, c.def, c.max); got != c.want {
			t.Errorf("ClampLimit(%d, %d, %d) = %d, want %d", c.req, c.def, c.max, got, c.want)
		}
	}
}
