package cmd

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestIsToday(t *testing.T) {
	tests := []struct {
		now    time.Time
		target time.Time
		want   bool
	}{
		{now: parseTime("2022-01-02T17:00:00+09:00"), target: parseTime("2022-01-01T23:59:59+09:00"), want: false},
		{now: parseTime("2022-01-02T17:00:00+09:00"), target: parseTime("2022-01-02T00:00:00+09:00"), want: true},
		{now: parseTime("2022-01-02T17:00:00+09:00"), target: parseTime("2022-01-01T14:59:59Z"), want: false},
		{now: parseTime("2022-01-02T17:00:00+09:00"), target: parseTime("2022-01-01T15:00:00Z"), want: true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("now:%s, target:%s", tt.now.String(), tt.target.String()), func(t *testing.T) {
			got := IsToday(tt.now, tt.target)
			if got != tt.want {
				t.Errorf("IsTarget() = %t; want %t", got, tt.want)
			}
		})
	}
}

func TestSortPullRequests(t *testing.T) {
	tests := []struct {
		pulls []PullRequest
		want  []PullRequest
	}{
		{
			pulls: []PullRequest{
				{Title: "TestPullRequest1", HtmlUrl: "http://example.com", CreatedAt: "2023-01-01T09:30:00Z"},
				{Title: "TestPullRequest2", HtmlUrl: "http://example.com", CreatedAt: "2023-01-01T09:00:00Z"},
			},
			want: []PullRequest{
				{Title: "TestPullRequest2", HtmlUrl: "http://example.com", CreatedAt: "2023-01-01T09:00:00Z"},
				{Title: "TestPullRequest1", HtmlUrl: "http://example.com", CreatedAt: "2023-01-01T09:30:00Z"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("pulls:%v", tt.pulls), func(t *testing.T) {
			got := SortPullRequests(tt.pulls)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SortPullRequests() = %v; want %v", got, tt.want)
			}
		})
	}
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
