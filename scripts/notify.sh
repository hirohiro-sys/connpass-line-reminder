#!/usr/bin/env bash
# connpass で参加登録した「明日」のイベントを LINE に送る。
# 公開 repo なので、ログにはユーザー名・イベント名・URL を出さない（件数と HTTP ステータスだけ）。
set -euo pipefail

: "${CONNPASS_API_KEY:?}"
: "${CONNPASS_NICKNAME:?}"
: "${LINE_CHANNEL_ACCESS_TOKEN:?}"
: "${LINE_USER_ID:?}"

readonly USER_AGENT="connpass-line-reminder"
readonly PAGE_SIZE=100
readonly MAX_PAGES=5
readonly LINE_TEXT_LIMIT=5000

# 明日の日付（JST）。手元で試すときは TARGET_DATE=YYYY-MM-DD で上書きできる。
target_date="${TARGET_DATE:-$(TZ=Asia/Tokyo date -d 'tomorrow' +%F)}"
ym="${target_date:0:4}${target_date:5:2}"

# 指定した月に参加するイベントを、全ページ分 JSON の配列で返す。
fetch_events() {
  local start=1 page available fetched=0 body
  local events='[]'
  for ((page = 0; page < MAX_PAGES; page++)); do
    ((page > 0)) && sleep 1 # connpass は 1 秒に 1 リクエストまで
    body="$(curl -sS --fail --max-time 20 \
      -H "X-API-Key: ${CONNPASS_API_KEY}" \
      -H "User-Agent: ${USER_AGENT}" \
      --get "https://connpass.com/api/v2/events/" \
      --data-urlencode "nickname=${CONNPASS_NICKNAME}" \
      --data-urlencode "ym=${ym}" \
      --data-urlencode "count=${PAGE_SIZE}" \
      --data-urlencode "start=${start}")" || {
      echo "connpass の取得に失敗した" >&2
      return 1
    }
    available="$(jq -er '.results_available' <<<"$body")"
    events="$(jq -c --argjson more "$(jq -c '.events' <<<"$body")" '. + $more' <<<"$events")"
    fetched="$(jq 'length' <<<"$events")"
    ((fetched >= available)) && break
    start=$((fetched + 1))
  done
  if ((fetched < available)); then
    echo "connpass の結果が取り切れなかった（${fetched}/${available}）" >&2
    return 1
  fi
  printf '%s' "$events"
}

events="$(fetch_events)"

# 中止を除き、開催日（JST）が明日のものを「時刻<TAB>タイトル<TAB>URL」にして、時刻順に並べる。
lines=""
count=0
while IFS=$'\t' read -r started title url; do
  [[ -z "$started" ]] && continue
  jst="$(TZ=Asia/Tokyo date -d "$started" '+%F %H:%M')" || continue
  [[ "${jst:0:10}" == "$target_date" ]] || continue
  lines+="${jst:11:5}"$'\t'"${title}"$'\t'"${url}"$'\n'
  count=$((count + 1))
done < <(jq -r '.[] | select(.open_status != "cancelled" and .started_at != null) | [.started_at, .title, .url] | @tsv' <<<"$events")

echo "対象日の件数: ${count}"
if ((count == 0)); then
  exit 0
fi

# 開始時刻が未定のイベントは取り込まない（started_at が null）ので、時刻は必ずある。
header="明日のconnpass ${count}件（$((10#${target_date:5:2}))/$((10#${target_date:8:2}))）"
text="$header"
while IFS=$'\t' read -r time title url; do
  text+=$'\n'"${time} ${title}"$'\n'"${url}"
done < <(printf '%s' "$lines" | sort -s -t $'\t' -k1,1)
text="${text:0:LINE_TEXT_LIMIT}"

if [[ "${DRY_RUN:-}" == "1" ]]; then
  printf '%s\n' "$text"
  exit 0
fi

status="$(curl -sS --max-time 20 -o /dev/null -w '%{http_code}' \
  -X POST "https://api.line.me/v2/bot/message/push" \
  -H "Authorization: Bearer ${LINE_CHANNEL_ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "$(jq -n --arg to "$LINE_USER_ID" --arg text "$text" \
    '{to: $to, messages: [{type: "text", text: $text}]}')")"
echo "LINE push: HTTP ${status}"
[[ "$status" == "200" ]]
