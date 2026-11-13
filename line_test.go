package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPushSendsTextMessage(t *testing.T) {
	var auth, ctype string
	var body struct {
		To       string `json:"to"`
		Messages []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"messages"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		ctype = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
	}))
	defer srv.Close()

	l := &LineClient{PushURL: srv.URL, Token: "tok", HTTP: srv.Client()}
	if err := l.Push(context.Background(), "Uxxx", "こんにちは"); err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer tok" || ctype != "application/json" {
		t.Fatalf("auth=%q content-type=%q", auth, ctype)
	}
	if body.To != "Uxxx" || len(body.Messages) != 1 || body.Messages[0].Type != "text" || body.Messages[0].Text != "こんにちは" {
		t.Fatalf("%+v", body)
	}
}

func TestPushFailsOnNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	l := &LineClient{PushURL: srv.URL, Token: "tok", HTTP: srv.Client()}
	if err := l.Push(context.Background(), "Uxxx", "x"); err == nil {
		t.Fatal("400 はエラー")
	}
}
