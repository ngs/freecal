package main

import (
	"reflect"
	"testing"
	"time"
)

func TestMustParseClock(t *testing.T) {
	tests := []struct {
		name  string
		input string
		wantH int
		wantM int
	}{
		{
			name:  "parse 09:00",
			input: "09:00",
			wantH: 9,
			wantM: 0,
		},
		{
			name:  "parse 17:30",
			input: "17:30",
			wantH: 17,
			wantM: 30,
		},
		{
			name:  "parse 00:00",
			input: "00:00",
			wantH: 0,
			wantM: 0,
		},
		{
			name:  "parse 23:59",
			input: "23:59",
			wantH: 23,
			wantM: 59,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotH, gotM := mustParseClock(tt.input)
			if gotH != tt.wantH || gotM != tt.wantM {
				t.Errorf("mustParseClock(%q) = (%d, %d), want (%d, %d)",
					tt.input, gotH, gotM, tt.wantH, tt.wantM)
			}
		})
	}
}

func TestFormatJpWeekday(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Tokyo")

	tests := []struct {
		name string
		date string
		want string
	}{
		{
			name: "Monday",
			date: "2025-01-13",
			want: "月",
		},
		{
			name: "Tuesday",
			date: "2025-01-14",
			want: "火",
		},
		{
			name: "Wednesday",
			date: "2025-01-15",
			want: "水",
		},
		{
			name: "Thursday",
			date: "2025-01-16",
			want: "木",
		},
		{
			name: "Friday",
			date: "2025-01-17",
			want: "金",
		},
		{
			name: "Saturday",
			date: "2025-01-18",
			want: "土",
		},
		{
			name: "Sunday",
			date: "2025-01-19",
			want: "日",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, _ := time.ParseInLocation("2006-01-02", tt.date, loc)
			got := formatJpWeekday(date)
			if got != tt.want {
				t.Errorf("formatJpWeekday(%s) = %q, want %q", tt.date, got, tt.want)
			}
		})
	}
}

func TestOverlaps(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Tokyo")

	parseTime := func(s string) time.Time {
		t, _ := time.ParseInLocation("2006-01-02 15:04", s, loc)
		return t
	}

	tests := []struct {
		name        string
		a           interval
		b           interval
		wantOverlap bool
		wantStart   string
		wantEnd     string
	}{
		{
			name: "complete overlap",
			a: interval{
				start: parseTime("2025-01-13 09:00"),
				end:   parseTime("2025-01-13 12:00"),
			},
			b: interval{
				start: parseTime("2025-01-13 10:00"),
				end:   parseTime("2025-01-13 11:00"),
			},
			wantOverlap: true,
			wantStart:   "2025-01-13 10:00",
			wantEnd:     "2025-01-13 11:00",
		},
		{
			name: "partial overlap",
			a: interval{
				start: parseTime("2025-01-13 09:00"),
				end:   parseTime("2025-01-13 11:00"),
			},
			b: interval{
				start: parseTime("2025-01-13 10:00"),
				end:   parseTime("2025-01-13 12:00"),
			},
			wantOverlap: true,
			wantStart:   "2025-01-13 10:00",
			wantEnd:     "2025-01-13 11:00",
		},
		{
			name: "no overlap - before",
			a: interval{
				start: parseTime("2025-01-13 09:00"),
				end:   parseTime("2025-01-13 10:00"),
			},
			b: interval{
				start: parseTime("2025-01-13 11:00"),
				end:   parseTime("2025-01-13 12:00"),
			},
			wantOverlap: false,
		},
		{
			name: "no overlap - after",
			a: interval{
				start: parseTime("2025-01-13 11:00"),
				end:   parseTime("2025-01-13 12:00"),
			},
			b: interval{
				start: parseTime("2025-01-13 09:00"),
				end:   parseTime("2025-01-13 10:00"),
			},
			wantOverlap: false,
		},
		{
			name: "adjacent intervals",
			a: interval{
				start: parseTime("2025-01-13 09:00"),
				end:   parseTime("2025-01-13 10:00"),
			},
			b: interval{
				start: parseTime("2025-01-13 10:00"),
				end:   parseTime("2025-01-13 11:00"),
			},
			wantOverlap: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := overlaps(tt.a, tt.b)
			if ok != tt.wantOverlap {
				t.Errorf("overlaps() overlap = %v, want %v", ok, tt.wantOverlap)
			}
			if ok && tt.wantOverlap {
				wantStart := parseTime(tt.wantStart)
				wantEnd := parseTime(tt.wantEnd)
				if !got.start.Equal(wantStart) || !got.end.Equal(wantEnd) {
					t.Errorf("overlaps() = {%v, %v}, want {%v, %v}",
						got.start, got.end, wantStart, wantEnd)
				}
			}
		})
	}
}

func TestMergeIntervals(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Tokyo")

	parseTime := func(s string) time.Time {
		t, _ := time.ParseInLocation("2006-01-02 15:04", s, loc)
		return t
	}

	tests := []struct {
		name string
		in   []interval
		want []interval
	}{
		{
			name: "empty intervals",
			in:   []interval{},
			want: nil,
		},
		{
			name: "single interval",
			in: []interval{
				{
					start: parseTime("2025-01-13 09:00"),
					end:   parseTime("2025-01-13 10:00"),
				},
			},
			want: []interval{
				{
					start: parseTime("2025-01-13 09:00"),
					end:   parseTime("2025-01-13 10:00"),
				},
			},
		},
		{
			name: "non-overlapping intervals",
			in: []interval{
				{
					start: parseTime("2025-01-13 09:00"),
					end:   parseTime("2025-01-13 10:00"),
				},
				{
					start: parseTime("2025-01-13 11:00"),
					end:   parseTime("2025-01-13 12:00"),
				},
			},
			want: []interval{
				{
					start: parseTime("2025-01-13 09:00"),
					end:   parseTime("2025-01-13 10:00"),
				},
				{
					start: parseTime("2025-01-13 11:00"),
					end:   parseTime("2025-01-13 12:00"),
				},
			},
		},
		{
			name: "overlapping intervals",
			in: []interval{
				{
					start: parseTime("2025-01-13 09:00"),
					end:   parseTime("2025-01-13 11:00"),
				},
				{
					start: parseTime("2025-01-13 10:00"),
					end:   parseTime("2025-01-13 12:00"),
				},
			},
			want: []interval{
				{
					start: parseTime("2025-01-13 09:00"),
					end:   parseTime("2025-01-13 12:00"),
				},
			},
		},
		{
			name: "adjacent intervals",
			in: []interval{
				{
					start: parseTime("2025-01-13 09:00"),
					end:   parseTime("2025-01-13 10:00"),
				},
				{
					start: parseTime("2025-01-13 10:00"),
					end:   parseTime("2025-01-13 11:00"),
				},
			},
			want: []interval{
				{
					start: parseTime("2025-01-13 09:00"),
					end:   parseTime("2025-01-13 11:00"),
				},
			},
		},
		{
			name: "multiple overlapping intervals",
			in: []interval{
				{
					start: parseTime("2025-01-13 09:00"),
					end:   parseTime("2025-01-13 10:30"),
				},
				{
					start: parseTime("2025-01-13 10:00"),
					end:   parseTime("2025-01-13 11:00"),
				},
				{
					start: parseTime("2025-01-13 10:45"),
					end:   parseTime("2025-01-13 12:00"),
				},
			},
			want: []interval{
				{
					start: parseTime("2025-01-13 09:00"),
					end:   parseTime("2025-01-13 12:00"),
				},
			},
		},
		{
			name: "unsorted intervals",
			in: []interval{
				{
					start: parseTime("2025-01-13 11:00"),
					end:   parseTime("2025-01-13 12:00"),
				},
				{
					start: parseTime("2025-01-13 09:00"),
					end:   parseTime("2025-01-13 10:00"),
				},
				{
					start: parseTime("2025-01-13 14:00"),
					end:   parseTime("2025-01-13 15:00"),
				},
			},
			want: []interval{
				{
					start: parseTime("2025-01-13 09:00"),
					end:   parseTime("2025-01-13 10:00"),
				},
				{
					start: parseTime("2025-01-13 11:00"),
					end:   parseTime("2025-01-13 12:00"),
				},
				{
					start: parseTime("2025-01-13 14:00"),
					end:   parseTime("2025-01-13 15:00"),
				},
			},
		},
		{
			name: "contained intervals",
			in: []interval{
				{
					start: parseTime("2025-01-13 09:00"),
					end:   parseTime("2025-01-13 12:00"),
				},
				{
					start: parseTime("2025-01-13 10:00"),
					end:   parseTime("2025-01-13 11:00"),
				},
			},
			want: []interval{
				{
					start: parseTime("2025-01-13 09:00"),
					end:   parseTime("2025-01-13 12:00"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeIntervals(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeIntervals() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOpenBrowser(t *testing.T) {
	// This test just ensures the function doesn't panic
	// It won't actually open a browser in test environment
	t.Run("openBrowser doesn't panic", func(t *testing.T) {
		// This should not panic even if browser opening fails
		openBrowser("http://example.com")
	})
}
