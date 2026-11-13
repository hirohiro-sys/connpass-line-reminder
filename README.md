# connpass-line-reminder

connpass で参加登録した「明日」のイベントを、毎日 20:00（JST）に LINE へ送る。Go 製で、GitHub Actions だけで動く（サーバーも DB も Docker も不要）。

## 仕組み

1. `.github/workflows/notify.yml` が毎日 11:00 UTC に、テスト（`go test`）のあと `go run .` を動かす
2. connpass API v2 でユーザー名の参加イベントを取る（明日の月だけ）
3. 中止を除き、開催日（JST）が明日のものを時刻順に LINE へ push する。0件なら送らない

状態は持たない。通知の時点の connpass の状態がそのまま反映される。

## 設定

Settings → Secrets and variables → Actions に、次の4つを登録する。公開 repo なので、ユーザー名も secret にする。

| 名前 | 内容 |
| --- | --- |
| `CONNPASS_API_KEY` | connpass API のキー |
| `CONNPASS_NICKNAME` | connpass のユーザー名 |
| `LINE_CHANNEL_ACCESS_TOKEN` | LINE Messaging API のチャネルアクセストークン |
| `LINE_USER_ID` | 送り先の LINE userId |

## 動作確認

Actions タブの `notify` から Run workflow で手動実行できる。

手元で、LINE には送らず文面だけ見る（Go が要る。LINE の2つの変数は不要）:

```bash
DRY_RUN=1 TARGET_DATE=2026-11-14 CONNPASS_API_KEY=... CONNPASS_NICKNAME=... go run .
```

テスト:

```bash
go test ./...
```

## ファイル

| ファイル | 役割 |
| --- | --- |
| `main.go` | 環境変数の読み込みと全体の流れ |
| `connpass.go` | connpass API の取得とページ送り |
| `message.go` | 明日の絞り込み・時刻順・本文づくり（JST） |
| `line.go` | LINE への push |

## 注意

- ログは公開されるため、プログラムはユーザー名・イベント名・URL を出さない
- 参加取り消し・補欠が connpass の検索結果にどう出るかは未確認
- 60 日間 repo に動きがないと、GitHub が scheduled workflow を止める。止まったら Actions タブから再開する
