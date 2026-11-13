// connpass で参加登録した「明日」のイベントを LINE に送る。
// 公開 repo の Actions ログに残るため、ユーザー名・イベント名・URL は出力しない（件数と HTTP ステータスだけ）。
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	if err := run(context.Background(), os.Getenv, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "失敗:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, getenv func(string) string, now time.Time) error {
	dryRun := getenv("DRY_RUN") == "1"

	need := []string{"CONNPASS_API_KEY", "CONNPASS_NICKNAME"}
	if !dryRun {
		need = append(need, "LINE_CHANNEL_ACCESS_TOKEN", "LINE_USER_ID")
	}
	for _, name := range need {
		if getenv(name) == "" {
			return fmt.Errorf("環境変数 %s が空", name)
		}
	}

	// 手元で試すときは TARGET_DATE=YYYY-MM-DD で上書きできる。
	date := getenv("TARGET_DATE")
	if date == "" {
		date = targetDate(now)
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return errors.New("TARGET_DATE は YYYY-MM-DD で指定する")
	}
	ym := strings.ReplaceAll(date[:7], "-", "")

	events, err := newConnpassClient(getenv("CONNPASS_API_KEY")).FetchAttended(ctx, getenv("CONNPASS_NICKNAME"), ym)
	if err != nil {
		return err
	}
	items, err := selectOn(events, date)
	if err != nil {
		return err
	}
	fmt.Printf("対象日の件数: %d\n", len(items))
	if len(items) == 0 {
		return nil
	}

	text := buildText(date, items)
	if dryRun {
		fmt.Println(text)
		return nil
	}
	if err := newLineClient(getenv("LINE_CHANNEL_ACCESS_TOKEN")).Push(ctx, getenv("LINE_USER_ID"), text); err != nil {
		return err
	}
	fmt.Println("LINE push: 成功")
	return nil
}
