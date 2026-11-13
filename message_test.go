package main

import (
	"strings"
	"testing"
	"time"
)

func ptr(s string) *string { return &s }

func TestTargetDate(t *testing.T) {
	cases := []struct {
		name string
		now  time.Time
		want string
	}{
		{"JST の夜", time.Date(2026, 11, 13, 20, 0, 0, 0, jst), "2026-11-14"},
		{"UTC の 11:00 は JST の 20:00", time.Date(2026, 11, 13, 11, 0, 0, 0, time.UTC), "2026-11-14"},
		{"UTC で日付が違っても JST で数える", time.Date(2026, 11, 13, 16, 0, 0, 0, time.UTC), "2026-11-15"},
		{"月末", time.Date(2026, 11, 30, 20, 0, 0, 0, jst), "2026-12-01"},
		{"年末", time.Date(2026, 12, 31, 20, 0, 0, 0, jst), "2027-01-01"},
	}
	for _, c := range cases {
		if got := targetDate(c.now); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}

func TestSelectTomorrow(t *testing.T) {
	events := []Event{
		{ID: 1, Title: "夜", URL: "u1", StartedAt: ptr("2026-11-14T19:00:00+09:00"), OpenStatus: "open"},
		{ID: 2, Title: "朝", URL: "u2", StartedAt: ptr("2026-11-14T09:30:00+09:00"), OpenStatus: "open"},
		{ID: 3, Title: "別の日", URL: "u3", StartedAt: ptr("2026-11-15T10:00:00+09:00"), OpenStatus: "open"},
		{ID: 4, Title: "中止", URL: "u4", StartedAt: ptr("2026-11-14T12:00:00+09:00"), OpenStatus: "cancelled"},
		{ID: 5, Title: "時刻未定", URL: "u5", StartedAt: nil, OpenStatus: "open"},
	}
	got, err := selectOn(events, "2026-11-14")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Title != "朝" || got[1].Title != "夜" {
		t.Fatalf("時刻順の2件になるはず: %+v", got)
	}
}

func TestSelectUsesJSTBoundary(t *testing.T) {
	events := []Event{
		// UTC では 11/13 だが、JST では 11/14 0:00。
		{ID: 1, Title: "JST 0時", URL: "u", StartedAt: ptr("2026-11-13T15:00:00Z"), OpenStatus: "open"},
		// JST の 23:59 は、まだ 11/14。
		{ID: 2, Title: "JST 23:59", URL: "u", StartedAt: ptr("2026-11-14T23:59:00+09:00"), OpenStatus: "open"},
		// JST の 11/15 0:00 は対象外。
		{ID: 3, Title: "翌日 0時", URL: "u", StartedAt: ptr("2026-11-15T00:00:00+09:00"), OpenStatus: "open"},
	}
	got, err := selectOn(events, "2026-11-14")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Title != "JST 0時" || got[1].Title != "JST 23:59" {
		t.Fatalf("JST で数えた2件になるはず: %+v", got)
	}
}

func TestSelectEmpty(t *testing.T) {
	got, err := selectOn(nil, "2026-11-14")
	if err != nil || len(got) != 0 {
		t.Fatalf("0件で成功するはず: %+v, %v", got, err)
	}
}

func TestSelectRejectsInvalidStartedAt(t *testing.T) {
	events := []Event{{ID: 1, Title: "x", URL: "u", StartedAt: ptr("2026-11-14 19:00"), OpenStatus: "open"}}
	if _, err := selectOn(events, "2026-11-14"); err == nil {
		t.Fatal("読めない日時はエラーにして、通知の取りこぼしを黙って起こさない")
	}
}

func TestBuildText(t *testing.T) {
	items := []Item{
		{Time: "09:30", Title: "朝", URL: "https://connpass.com/event/2/"},
		{Time: "19:00", Title: "夜", URL: "https://connpass.com/event/1/"},
	}
	want := "明日のconnpass 2件（11/14）\n" +
		"09:30 朝\nhttps://connpass.com/event/2/\n" +
		"19:00 夜\nhttps://connpass.com/event/1/"
	if got := buildText("2026-11-14", items); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildTextTruncatesToLineLimit(t *testing.T) {
	var items []Item
	for i := 0; i < 400; i++ {
		items = append(items, Item{Time: "10:00", Title: strings.Repeat("あ", 20), URL: "https://connpass.com/event/1/"})
	}
	got := buildText("2026-11-14", items)
	if n := len([]rune(got)); n > lineTextLimit {
		t.Fatalf("LINE の上限 %d 文字を超えた: %d", lineTextLimit, n)
	}
}
