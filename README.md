# connpass-line-reminder

connpass で参加登録した「明日」のイベントを、毎日 20:00（JST）に LINE へ送る。GitHub Actions だけで動く（サーバーも DB も不要）。

## 仕組み

1. `.github/workflows/notify.yml` が毎日 11:00 UTC に `scripts/notify.sh` を動かす
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

手元で、送らずに文面だけ見る（`jq` と GNU `date` が要る）:

```bash
DRY_RUN=1 TARGET_DATE=2026-11-14 \
CONNPASS_API_KEY=... CONNPASS_NICKNAME=... LINE_CHANNEL_ACCESS_TOKEN=x LINE_USER_ID=x \
./scripts/notify.sh
```

## 注意

- ログは公開されるため、スクリプトはユーザー名・イベント名・URL を出さない
- 参加取り消し・補欠が connpass の検索結果にどう出るかは未確認
- 60 日間 repo に動きがないと、GitHub が scheduled workflow を止める。止まったら Actions タブから再開する
