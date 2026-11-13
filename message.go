package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// 通知は JST の日付で数える。
var jst = time.FixedZone("JST", 9*60*60)

// LINE のテキストメッセージは 5000 文字まで。
const lineTextLimit = 5000

// Event は connpass API v2 のイベントのうち、通知に使う項目。
type Event struct {
	ID         int     `json:"id"`
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	StartedAt  *string `json:"started_at"`
	OpenStatus string  `json:"open_status"`
}

// Item は通知に載せる1件。
type Item struct {
	Time  string // JST の HH:MM
	Title string
	URL   string
}

// targetDate は、now の翌日の日付（JST、YYYY-MM-DD）。
func targetDate(now time.Time) string {
	return now.In(jst).AddDate(0, 0, 1).Format("2006-01-02")
}

// selectOn は、date（JST の YYYY-MM-DD）に開催される、中止でないイベントを時刻順に返す。
// 開始日時が未定のイベントは含めない。読めない日時があればエラーにする。
func selectOn(events []Event, date string) ([]Item, error) {
	type dated struct {
		at   time.Time
		item Item
	}
	var picked []dated
	for _, e := range events {
		if e.OpenStatus == "cancelled" || e.StartedAt == nil {
			continue
		}
		at, err := time.Parse(time.RFC3339, *e.StartedAt)
		if err != nil {
			return nil, fmt.Errorf("開始日時の形が合わないイベントがある（id=%d）", e.ID)
		}
		local := at.In(jst)
		if local.Format("2006-01-02") != date {
			continue
		}
		picked = append(picked, dated{at: at, item: Item{Time: local.Format("15:04"), Title: e.Title, URL: e.URL}})
	}
	sort.SliceStable(picked, func(i, j int) bool { return picked[i].at.Before(picked[j].at) })
	items := make([]Item, len(picked))
	for i, p := range picked {
		items[i] = p.item
	}
	return items, nil
}

// buildText は LINE に送る本文。上限を超えたら、後ろを切る。
func buildText(date string, items []Item) string {
	t, _ := time.Parse("2006-01-02", date)
	var b strings.Builder
	fmt.Fprintf(&b, "明日のconnpass %d件（%d/%d）", len(items), int(t.Month()), t.Day())
	for _, it := range items {
		fmt.Fprintf(&b, "\n%s %s\n%s", it.Time, it.Title, it.URL)
	}
	runes := []rune(b.String())
	if len(runes) > lineTextLimit {
		runes = runes[:lineTextLimit]
	}
	return string(runes)
}
