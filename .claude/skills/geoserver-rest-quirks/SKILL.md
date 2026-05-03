---
description: GeoServer 2.27 / 2.28 REST API quirks that this client works around — workspace-scoped POST /styles Accept-header dispatch, mixed-shape JSON arrays, empty-collection string-vs-object payloads, version-specific endpoint differences. Reference content; loaded automatically when working on REST code or debugging GeoServer responses.
when_to_use: User mentions "GeoServer returned 500", "unmarshal error", "unexpected JSON shape", "Accept header", "why does this style endpoint behave differently", "no attributes", or is implementing a new REST endpoint client.
---

# GeoServer 2.x REST API quirks

This is reference material. Each quirk lists the symptom, the root cause as understood, and the file:line in this repo where the workaround lives. Whenever you change any code that talks to GeoServer, scan this list for relevance.

## 1. Workspace-scoped `POST /workspaces/{ws}/styles` requires `Accept: */*`

- **Symptom:** GeoServer 2.28 returns `500 "No such style handler: format = application/json"` if you send `Accept: application/json`.
- **Root cause:** GeoServer dispatches on the Accept header looking for a "style format handler" that matches that media type. There is no JSON style handler, so it 500s.
- **Workaround:** Send `Accept: "*/*"` to disable the dispatch and route to the metadata-creation path. The body's `Content-Type` should also be `application/json; charset=utf-8` — bare `application/json` 500s in some older 2.x versions.
- **Where:** `styles.go:178-186` (`CreateStyleContext`).

## 2. Empty styles collection comes back as `{"styles":""}` (bare string)

- **Symptom:** `GetStyles` against a fresh / empty workspace fails to unmarshal because GeoServer returns a JSON object whose `styles` field is the empty string instead of an object.
- **Workaround:** Decode into `json.RawMessage` first; branch on the first byte (`'"'` ⇒ empty list; `'{'` ⇒ decode as `{"style": [...]}`).
- **Where:** `styles.go:93-104` (`GetStylesContext`).

## 3. `LayerGroup.styles.style` is a mixed `[string|object]` array

- **Symptom:** `GET /layergroups/{name}` for any layer group with default-styled members produces `"styles": {"style": ["", "", {...}, ""]}` — string entries (often empty) interspersed with style objects. Standard JSON decoder errors with `cannot unmarshal string into Go struct field LayerGroupStyles.style`.
- **Workaround:** Custom `UnmarshalJSON` on `LayerGroupStyles` that decodes the inner `style` array via `[]json.RawMessage`, then per-element handles `'"'` (string) vs `'{'` (object) and produces `[]*Resource`. String entries are stored as `&Resource{Name: stringValue}` to preserve the `[]*Resource` field type.
- **Where:** `layergroups.go` `LayerGroupStyles.UnmarshalJSON`.

## 4. POST style endpoints need explicit `; charset=utf-8`

- **Symptom:** Bare `Content-Type: application/json` 500s in some 2.x versions.
- **Workaround:** Send `Content-Type: application/json; charset=utf-8`.
- **Where:** Same site as quirk #1 — `styles.go:189` (`DataType: jsonType + "; charset=utf-8"`).

## 5. PostGIS publish requires the table to exist with attributes

- **Symptom:** `POST /workspaces/{ws}/datastores/{ds}/featuretypes` returns `400 "no attributes"` if the named table is empty or doesn't exist.
- **Workaround:** Tests bootstrap a real PostGIS table via `docker/postgis/init/01-lbldyt.sql` (creates `public.lbldyt(gid, name, label, geom)` with sample rows + GIST index). Production callers must ensure their target table exists.
- **Where:** `docker/postgis/init/01-lbldyt.sql`; tested in `layers_test.go` (integration).

## 6. `Settings.contact` is `interface{}` in v1

- **Symptom:** Type assertions on `Settings.Contact` panic in v1.0; in v1.1 they return zero values silently.
- **Root cause:** v1.0 modeled the contact subdocument loosely. `Contact` *type* is defined in `settings.go:15` but never wired up; the field is `interface{}` (`settings.go:27`).
- **Workaround:** Don't trust the field type for new code. v2 will make this concrete.
- **Where:** `settings.go:27`.

## 7. Pagination drift across versions

- **Symptom:** `GET /rest/layers` and `GET /rest/styles` paginate via `?startIndex=&count=` on GeoServer 2.18+ but return everything on older versions.
- **Workaround:** Send pagination params and tolerate them being ignored on older servers. v2 wraps this in `iter.Seq2[T, error]` with single-page fallback.
- **Where:** Not currently mitigated in v1; documented for awareness.

## 8. `ParseURL` must escape per segment, not the whole path

- **Symptom:** Workspace / layer names with spaces, slashes, or non-ASCII characters produced malformed URLs in v1.0.
- **Workaround:** `g.ParseURL(parts...)` applies `url.PathEscape` to each segment before `path.Join`. Use multi-arg `ParseURL("rest", "workspaces", ws, "styles")` instead of `fmt.Sprintf("%srest/%s/styles", server, ws)`.
- **Where:** `utils.go` `ParseURL`.

## 9. Empty `wfs:FeatureType` lists in capabilities

- **Symptom:** Older GeoServer versions emit `<FeatureTypeList></FeatureTypeList>` (empty) while newer ones omit the element entirely.
- **Workaround:** XML parser treats both as empty list; no caller-visible difference. WMS capabilities parser uses `wms.ParseCapabilitiesE` which returns `(*Capabilities, error)` — prefer it over the deprecated `ParseCapabilities` that swallowed errors.
- **Where:** `wms/wms.go` `ParseCapabilitiesE`.

---

## When this skill is most useful

- Implementing a new REST resource client — scan the list before writing the request to avoid re-discovering quirk 1, 4, or 8.
- Debugging an integration-test failure with `unmarshal` or `5xx` in the message.
- Reading a PR that touches `styles.go`, `layergroups.go`, or `utils.go` `ParseURL` — verify the quirk-handling code wasn't naively "simplified" away.
