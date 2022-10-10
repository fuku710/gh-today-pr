package main

import (
	"fmt"
	"testing"
	"time"
)

func TestIn24hours(t *testing.T) {
	tests := []struct {
		now    time.Time
		target time.Time
		want   bool
	}{
		{now: ParseTime("2022-01-03T00:00:00"), target: ParseTime("2022-01-01T23:59:59"), want: false},
		{now: ParseTime("2022-01-03T00:00:00"), target: ParseTime("2022-01-02T00:00:00"), want: true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("now:%s, target:%s", tt.now.String(), tt.target.String()), func(t *testing.T) {
			got := In24hours(tt.now, tt.target)
			if got != tt.want {
				t.Errorf("In24Hours() = %t; want true", tt.want)
			}
		})
	}

}

func ParseTime(s string) time.Time {
	t, _ := time.Parse("2006-01-02T15:04:05", s)
	return t
}
