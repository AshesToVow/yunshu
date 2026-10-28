package system

import (
	"strings"
	"testing"
	"time"
)

// TestResolveOperationLogWindow 表驱动覆盖时间窗解析：默认窗、单边缺省、日期粒度、非法值与跨度上限。
func TestResolveOperationLogWindow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.Local)

	cases := []struct {
		name      string
		startRaw  string
		endRaw    string
		check     func(t *testing.T, start, end time.Time, err error)
	}{
		{
			name: "both empty defaults to last 7 days",
			check: func(t *testing.T, start, end time.Time, err error) {
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				if !start.Equal(now.AddDate(0, 0, -7)) || !end.Equal(now) {
					t.Fatalf("start=%v end=%v, want %v..%v", start, end, now.AddDate(0, 0, -7), now)
				}
			},
		},
		{
			name:     "start only fills end with now",
			startRaw: "2026-10-20 08:00:00",
			check: func(t *testing.T, start, end time.Time, err error) {
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				if want := time.Date(2026, 10, 20, 8, 0, 0, 0, time.Local); !start.Equal(want) {
					t.Fatalf("start=%v, want %v", start, want)
				}
				if !end.Equal(now) {
					t.Fatalf("end=%v, want %v", end, now)
				}
			},
		},
		{
			name:  "end only fills start 7 days back",
			endRaw: "2026-10-25",
			check: func(t *testing.T, start, end time.Time, err error) {
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				if wantEnd := time.Date(2026, 10, 25, 23, 59, 59, 0, time.Local); !end.Equal(wantEnd) {
					t.Fatalf("end=%v, want end-of-day %v", end, wantEnd)
				}
				if wantStart := time.Date(2026, 10, 18, 23, 59, 59, 0, time.Local); !start.Equal(wantStart) {
					t.Fatalf("start=%v, want %v", start, wantStart)
				}
			},
		},
		{
			name:     "both set keeps exact window",
			startRaw: "2026-10-01",
			endRaw:   "2026-10-28 09:30:00",
			check: func(t *testing.T, start, end time.Time, err error) {
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				if want := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local); !start.Equal(want) {
					t.Fatalf("start=%v, want %v", start, want)
				}
				if want := time.Date(2026, 10, 28, 9, 30, 0, 0, time.Local); !end.Equal(want) {
					t.Fatalf("end=%v, want %v", end, want)
				}
			},
		},
		{
			name:     "start after end rejected",
			startRaw: "2026-10-20",
			endRaw:   "2026-10-10",
			check: func(t *testing.T, _, _ time.Time, err error) {
				if err == nil || !strings.Contains(err.Error(), "开始时间不能晚于结束时间") {
					t.Fatalf("want start-after-end error, got %v", err)
				}
			},
		},
		{
			name:     "span beyond 92 days rejected",
			startRaw: "2026-01-01",
			endRaw:   "2026-10-28",
			check: func(t *testing.T, _, _ time.Time, err error) {
				if err == nil || !strings.Contains(err.Error(), "92") {
					t.Fatalf("want span-cap error, got %v", err)
				}
			},
		},
		{
			name:     "bad format rejected",
			startRaw: "2026/10/01",
			check: func(t *testing.T, _, _ time.Time, err error) {
				if err == nil || !strings.Contains(err.Error(), "时间格式") {
					t.Fatalf("want format error, got %v", err)
				}
			},
		},
		{
			name:  "end exactly 92 days after start allowed",
			startRaw: "2026-07-28 12:00:00",
			endRaw:   "2026-10-28 12:00:00",
			check: func(t *testing.T, _, _ time.Time, err error) {
				if err != nil {
					t.Fatalf("92-day window should be allowed, got %v", err)
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			start, end, err := resolveOperationLogWindow(tc.startRaw, tc.endRaw, now)
			tc.check(t, start, end, err)
		})
	}
}
