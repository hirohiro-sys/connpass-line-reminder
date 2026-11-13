package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const linePushURL = "https://api.line.me/v2/bot/message/push"

// LineClient は LINE Messaging API の push の呼び出し口。
type LineClient struct {
	PushURL string
	Token   string
	HTTP    *http.Client
}

func newLineClient(token string) *LineClient {
	return &LineClient{PushURL: linePushURL, Token: token, HTTP: &http.Client{Timeout: 20 * time.Second}}
}

// Push は text を to（LINE の userId）に送る。エラーに本文や宛先を含めない（公開ログに出るため）。
func (l *LineClient) Push(ctx context.Context, to, text string) error {
	payload, err := json.Marshal(map[string]any{
		"to":       to,
		"messages": []map[string]string{{"type": "text", "text": text}},
	})
	if err != nil {
		return errors.New("LINE の本文を作れなかった")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.PushURL, bytes.NewReader(payload))
	if err != nil {
		return errors.New("LINE へのリクエストを作れなかった")
	}
	req.Header.Set("Authorization", "Bearer "+l.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.HTTP.Do(req)
	if err != nil {
		return errors.New("LINE に接続できなかった")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LINE が HTTP %d を返した", resp.StatusCode)
	}
	return nil
}
