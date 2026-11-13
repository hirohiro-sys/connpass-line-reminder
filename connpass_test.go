package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func newTestClient(srv *httptest.Server) *ConnpassClient {
	return &ConnpassClient{
		BaseURL: srv.URL,
		APIKey:  "key",
		HTTP:    srv.Client(),
		Sleep:   func(time.Duration) {},
	}
}

func TestFetchSendsRequiredParams(t *testing.T) {
	var gotKey, gotUA, gotNick, gotYM string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-API-Key")
		gotUA = r.Header.Get("User-Agent")
		gotNick = r.URL.Query().Get("nickname")
		gotYM = r.URL.Query().Get("ym")
		fmt.Fprint(w, `{"results_available":0,"events":[]}`)
	}))
	defer srv.Close()

	if _, err := newTestClient(srv).FetchAttended(context.Background(), "taro", "202611"); err != nil {
		t.Fatal(err)
	}
	if gotKey != "key" || gotUA == "" || gotNick != "taro" || gotYM != "202611" {
		t.Fatalf("key=%q ua=%q nickname=%q ym=%q", gotKey, gotUA, gotNick, gotYM)
	}
}

func TestFetchFollowsPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start, _ := strconv.Atoi(r.URL.Query().Get("start"))
		// 全 3 件を、1ページ 2 件ずつ返す。
		var body string
		switch start {
		case 1:
			body = `{"results_available":3,"events":[{"id":1,"title":"a","url":"u","started_at":null,"open_status":"open"},{"id":2,"title":"b","url":"u","started_at":null,"open_status":"open"}]}`
		case 3:
			body = `{"results_available":3,"events":[{"id":3,"title":"c","url":"u","started_at":null,"open_status":"open"}]}`
		default:
			t.Errorf("想定外の start=%d", start)
		}
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	events, err := newTestClient(srv).FetchAttended(context.Background(), "taro", "202611")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("3件になるはず: %d", len(events))
	}
}

func TestFetchFailsOnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := newTestClient(srv).FetchAttended(context.Background(), "taro", "202611")
	if err == nil {
		t.Fatal("429 はエラーになるはず")
	}
}

func TestFetchErrorDoesNotLeakNickname(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // 接続できない状態にする

	c := &ConnpassClient{BaseURL: srv.URL, APIKey: "key", HTTP: http.DefaultClient, Sleep: func(time.Duration) {}}
	_, err := c.FetchAttended(context.Background(), "secret-nickname", "202611")
	if err == nil {
		t.Fatal("接続できなければエラー")
	}
	if msg := err.Error(); contains(msg, "secret-nickname") {
		t.Fatalf("公開ログにユーザー名が出る: %s", msg)
	}
}

func TestFetchFailsWhenIncomplete(t *testing.T) {
	// 件数が残っているのに、空のページが返る。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"results_available":5,"events":[]}`)
	}))
	defer srv.Close()

	if _, err := newTestClient(srv).FetchAttended(context.Background(), "taro", "202611"); err == nil {
		t.Fatal("取り切れなければエラーにする（一部だけで通知しない）")
	}
}

func TestFetchFailsOnBrokenJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `not json`)
	}))
	defer srv.Close()

	if _, err := newTestClient(srv).FetchAttended(context.Background(), "taro", "202611"); err == nil {
		t.Fatal("壊れた応答はエラー")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
