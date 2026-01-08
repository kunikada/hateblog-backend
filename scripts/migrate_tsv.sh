#!/usr/bin/env sh
set -eu

bookmarks_tsv="bookmarks.tsv"
keywords_tsv="keywords.tsv"
keyphrases_tsv="keyphrases.tsv"

entries_tsv="entries.tsv"
tags_tsv="tags.tsv"
entry_tags_tsv="entry_tags.tsv"

for file in "$bookmarks_tsv" "$keywords_tsv" "$keyphrases_tsv"; do
  if [ ! -f "$file" ]; then
    echo "missing input: $file" >&2
    exit 1
  fi
done

uuid_mode=""
if command -v uuidgen >/dev/null 2>&1; then
  uuid_mode="uuidgen"
elif [ -r /proc/sys/kernel/random/uuid ]; then
  uuid_mode="proc"
else
  echo "uuid generator not found (uuidgen or /proc/sys/kernel/random/uuid)" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT

bm_map="$tmpdir/bookmark_map.tsv"
kw_map="$tmpdir/keyword_map.tsv"
printf "bookmark_id\tentry_id\n" > "$bm_map"
printf "keyword_id\ttag_id\n" > "$kw_map"

now="$(TZ=UTC date +'%Y-%m-%d %H:%M:%S+00')"

TZ=UTC awk -F '\t' -v OFS='\t' -v bm_map="$bm_map" -v uuid_mode="$uuid_mode" '
function get_uuid(  u) {
  if (uuid_mode == "uuidgen") {
    "uuidgen" | getline u
    close("uuidgen")
  } else {
    getline u < "/proc/sys/kernel/random/uuid"
    close("/proc/sys/kernel/random/uuid")
  }
  return tolower(u)
}
function ts(u) {
  if (u == "\\N" || u == "") {
    return "\\N"
  }
  return strftime("%Y-%m-%d %H:%M:%S+00", u)
}
BEGIN {
  print "id\ttitle\turl\tposted_at\tbookmark_count\texcerpt\tsubject\tcreated_at\tupdated_at"
}
NR == 1 { next }
{
  id = get_uuid()
  old_id = $1
  title = $2
  link = $3
  sslp = $4
  description = $5
  subject = $6
  cnt = $7
  ientried = $8
  icreated = $9
  imodified = $10

  scheme = (sslp == "1") ? "https" : "http"
  url = scheme "://" link

  if (cnt == "\\N" || cnt == "") {
    cnt = 0
  }
  if (description == "") {
    description = "\\N"
  }
  if (subject == "") {
    subject = "\\N"
  }

  print id, title, url, ts(ientried), cnt, description, subject, ts(icreated), ts(imodified)
  print old_id "\t" id >> bm_map
}
' "$bookmarks_tsv" > "$entries_tsv"

TZ=UTC awk -F '\t' -v OFS='\t' -v kw_map="$kw_map" -v uuid_mode="$uuid_mode" -v now="$now" '
function get_uuid(  u) {
  if (uuid_mode == "uuidgen") {
    "uuidgen" | getline u
    close("uuidgen")
  } else {
    getline u < "/proc/sys/kernel/random/uuid"
    close("/proc/sys/kernel/random/uuid")
  }
  return tolower(u)
}
BEGIN {
  print "id\tname\tcreated_at"
}
NR == 1 { next }
{
  id = get_uuid()
  keyword_id = $1
  name = $2

  if (name == "\\N" || name == "") {
    print "keyword is null for id=" keyword_id > "/dev/stderr"
    exit 1
  }

  print id, name, now
  print keyword_id "\t" id >> kw_map
}
' "$keywords_tsv" > "$tags_tsv"

TZ=UTC awk -F '\t' -v OFS='\t' -v bm_map="$bm_map" -v kw_map="$kw_map" -v now="$now" '
FILENAME == bm_map && FNR > 1 {
  bm[$1] = $2
  next
}
FILENAME == kw_map && FNR > 1 {
  kw[$1] = $2
  next
}
FNR == 1 {
  print "entry_id\ttag_id\tscore\tcreated_at"
  next
}
{
  entry_id = bm[$1]
  tag_id = kw[$2]
  score = $3

  if (score == "\\N" || score == "") {
    score = 0
  }
  if (entry_id == "" || tag_id == "") {
    print "missing mapping for bookmark_id=" $1 ", keyword_id=" $2 > "/dev/stderr"
    exit 1
  }

  print entry_id, tag_id, score, now
}
' "$bm_map" "$kw_map" "$keyphrases_tsv" > "$entry_tags_tsv"
