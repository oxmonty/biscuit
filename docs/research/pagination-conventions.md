# Pagination conventions in the wild

2026-07-30. Survey seeding E5's built-in pagination scheme table (see [PRD — API semantics](../../PRD.md#api-semantics-handled-automatically)). Facts sourced from official docs; section 3 is the ranked recommendation the built-in library implements.

## 1. Schemes by type

### Cursor (opaque token, non-URL)

| API | Request param(s) | Response items field | Next-signal field | Next-signal kind | Source |
|---|---|---|---|---|---|
| Stripe | `starting_after`, `ending_before` (+ `limit`) | `data` | `has_more` | boolean | https://docs.stripe.com/api/pagination |
| OpenAI | `after` (+ `limit`) | `data` | `has_more` (+ `last_id`) | boolean, paired with a separate last-id field | https://developers.openai.com/api/reference |
| Slack | `cursor` (+ `limit`) | resource-specific (`members`, `channels`, `messages`) | `response_metadata.next_cursor` | string cursor value, nested one level | https://docs.slack.dev/apis/web-api/pagination/ |
| AWS (generic) | `NextToken` (varies: `NextPageToken`, `NextMarker`) | resource-specific (e.g. `Items`) | `NextToken` (same name reused in response) | opaque string token | https://docs.aws.amazon.com/ec2/latest/devguide/ec2-api-pagination.html |
| Google Cloud / AIP-158 | `pageToken` / `page_token` (+ `pageSize`/`page_size`) | resource-specific plural (`items`, `buckets`) | `nextPageToken` / `next_page_token` | opaque string token, empty/absent = end | https://google.aip.dev/158 |
| Shopify | `page_info` (+ `limit`) | resource-specific (`products`, `customers`) | `Link` header `rel="next"` embedding `page_info` | Link header (URL) | https://shopify.dev/docs/api/admin-rest/usage/pagination |
| JSON:API cursor profile | `page[cursor]` | `data` | `links.next` | full URL | https://jsonapi.org/profiles/ethanresnick/cursor-pagination/ |

### Cursor-in-URL / Link header

| API | Request param(s) | Response items field | Next-signal field | Next-signal kind | Source |
|---|---|---|---|---|---|
| GitHub REST | `page`, `per_page` (some endpoints `before`/`after`) | top-level array, no wrapper | `Link` response header `rel="next"` | full URL in an HTTP header | https://docs.github.com/en/rest/using-the-rest-api/using-pagination-in-the-rest-api |
| Twilio | `PageSize`, `Page`, `AfterSid` | resource-specific (`calls`) | `next_page_uri` (body, relative URL) | URL string in body | https://www.twilio.com/docs/usage/twilios-response |
| HAL | implementation-specific (often `page`/`size`) | `_embedded.<resource>` | `_links.next.href` | full URL under `_links` | https://apigility.org/documentation/api-primer/halprimer |

### Offset-based

| API | Request param(s) | Response items field | Next-signal field | Next-signal kind | Source |
|---|---|---|---|---|---|
| Classic offset/limit (DRF and similar) | `limit`, `offset` | `results` | `next`, `previous` | full URL or null | https://www.django-rest-framework.org/api-guide/pagination/ |

### Page-number-based

| API | Request param(s) | Response items field | Next-signal field | Next-signal kind | Source |
|---|---|---|---|---|---|
| Classic page/per_page (Rails-style; JSON:API `page[number]`/`page[size]`) | `page`, `per_page` | varies / `data` | `links.next` (JSON:API mandates `first`/`last`/`prev`/`next`) | full URL | https://jsonapi.org/format/ |
| GitHub (page mode) | `page`, `per_page` | top-level array | `Link` header `rel="next"` | header URL | as above |

## 2. Names that must never trigger alone

- **`limit`** — rate limits, truncation caps, batch sizes; shared across all schemes. Never sufficient.
- **`page`** — whole-param match only (`homepage`, `page_content`, `webpage_url` are false herrings); even whole, ambiguous between page-number and CMS-style content params.
- **`next`** — generic workflow/state field; only trust inside a recognized envelope (`links.next`, `response_metadata.next_cursor`, `_links.next`).
- **`offset`** — byte/seek/UTC offsets; needs `limit` co-occurrence plus an items array.
- **`cursor`** — DB/editor/replication cursors exist; corroborate.
- **`token`** — auth/CSRF/idempotency collisions; anchor literal names (`NextToken` et al.), never `*token*`.
- **`has_more`** — safest response-side signal, but still requires items-array corroboration.
- **`meta`** — generic envelope; only its specific nested keys signal pagination.

## 3. Recommended built-in set (by prevalence, near-zero false positives under both-sides corroboration)

1. **offset/limit** — req `offset`+`limit`; resp items array + `next`/`previous` URLs or count. (DRF and hand-rolled REST.)
2. **page/per_page** — req `page`+`per_page`|`page_size`; resp items array + `Link` header or `links.next`/`total_pages`.
3. **cursor + has_more (Stripe/OpenAI-style)** — req `starting_after`|`ending_before`|`after`|`cursor`; resp items array (`data`) + `has_more`.
4. **cursor + next_cursor (Slack-style)** — req `cursor`; resp items array + nested `next_cursor`.
5. **pageToken (Google/AIP-158)** — req `pageToken`/`page_token`; resp items array + `nextPageToken`/`next_page_token`.
6. **NextToken (AWS-style)** — req `NextToken`/`NextPageToken`/`NextMarker`; resp items array + same-name token field. Widest alias list; still exact whole-word.
7. **Link header (RFC 5988)** — resp items array + documented `Link` response header with `rel="next"`; detection must read the spec's response `headers:` section, not just body schema (GitHub's spec documents it).
8. *(optional)* **cursor-URL in body (Twilio-style)** — resp items array + `next_page_uri`.

JSON:API's `page[...]` family is a query-param namespacing overlay on schemes 1–2; HAL's `_links.next.href` is scheme 7's pattern in the body — fold both in as naming variants, not new schemes.

## 4. Ambiguity and collision handling

- **`page` + cursor both present** (transitional APIs; Shopify's page→cursor migration): treat cursor as authoritative, `page` as legacy.
- **`limit`+`offset` vs `limit`+`starting_after`**: discriminate on the second param, never on `limit`.
- **Multiple "next" response dialects**: match primarily on the request-side param; response side corroborates, never discriminates alone.
- **Link header vs body-URL**: same intent, different transport location; check both response body schema and documented response headers.
- **AWS naming drift**: one scheme, small exact-name alias set on both sides.
