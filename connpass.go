package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	connpassBaseURL = "https://connpass.com/api/v2"
	pageSize        = 100
	maxPages        = 5
	// connpass は 1 キーにつき 1 秒に 1 リクエストまで。
	requestInterval = time.Second
	requestTimeout  = 20 * time.Second
	// connpass は User-Agent の無いリクエストを 403 で弾く。
	userAgent = "connpass-line-reminder"
)

// ConnpassClient は connpass API v2 の呼び出し口。
type ConnpassClient struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
	Sleep   func(time.Duration)
}

func newConnpassClient(apiKey string) *ConnpassClient {
	return &ConnpassClient{
		BaseURL: connpassBaseURL,
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: requestTimeout},
		Sleep:   time.Sleep,
	}
}

type eventsPage struct {
	ResultsAvailable int     `json:"results_available"`
	Events           []Event `json:"events"`
}

// FetchAttended は、nickname が参加するイベントを、ym（YYYYMM）の月の分だけ全ページ取る。
// 取り切れなければエラーにする。公開ログに出るため、エラーにユーザー名や URL を含めない。
func (c *ConnpassClient) FetchAttended(ctx context.Context, nickname, ym string) ([]Event, error) {
	var events []Event
	available := 0
	for page := 0; page < maxPages; page++ {
		if page > 0 {
			c.Sleep(requestInterval)
		}
		p, err := c.getPage(ctx, nickname, ym, len(events)+1)
		if err != nil {
			return nil, err
		}
		available = p.ResultsAvailable
		events = append(events, p.Events...)
		if len(events) >= available {
			return events, nil
		}
		if len(p.Events) == 0 {
			break
		}
	}
	return nil, fmt.Errorf("connpass の結果を取り切れなかった（%d/%d）", len(events), available)
}

func (c *ConnpassClient) getPage(ctx context.Context, nickname, ym string, start int) (*eventsPage, error) {
	q := url.Values{}
	q.Set("nickname", nickname)
	q.Set("ym", ym)
	q.Set("count", strconv.Itoa(pageSize))
	q.Set("start", strconv.Itoa(start))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/events/?"+q.Encode(), nil)
	if err != nil {
		return nil, errors.New("connpass へのリクエストを作れなかった")
	}
	req.Header.Set("X-API-Key", c.APIKey)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		// err の文字列には URL（ユーザー名）が入るので、使わない。
		return nil, errors.New("connpass に接続できなかった")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("connpass が HTTP %d を返した", resp.StatusCode)
	}
	var p eventsPage
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, errors.New("connpass の応答を読めなかった")
	}
	return &p, nil
}
