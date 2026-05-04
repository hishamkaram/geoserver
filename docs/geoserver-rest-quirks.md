# GeoServer 2.x REST API quirks

This is the public version of the project's internal quirks catalog (mirrored from `.claude/skills/geoserver-rest-quirks/SKILL.md`). Each entry lists the symptom, the root cause as understood, and the file:line in this repo where the workaround lives.

If you're implementing a new REST resource client or debugging an integration-test failure with `unmarshal` or `5xx` in the message, scan this list first.

## 1. Workspace-scoped `POST /workspaces/{ws}/styles` requires `Accept: */*`

**Symptom:** GeoServer 2.28 returns `500 "No such style handler: format = application/json"` if you send `Accept: application/json`.

**Root cause:** GeoServer dispatches on the `Accept` header looking for a "style format handler" matching that media type. There's no JSON style handler — only SLD, ZIP, etc. — so the request 500s.

**Workaround:** Send `Accept: "*/*"` to disable the dispatch and route to the metadata-creation path. The body's `Content-Type` should also be `application/json; charset=utf-8` — bare `application/json` 500s in some older 2.x versions.

**Where:** `rest/styles/styles.go:141-173` (`Create` — workspace-scoped branch sets `accept = "*/*"`).

## 2. Empty styles collection comes back as `{"styles":""}` (bare string)

**Symptom:** `GetStyles` against a fresh / empty workspace fails to unmarshal because GeoServer returns a JSON object whose `styles` field is the empty string instead of an object.

**Workaround:** Decode into `json.RawMessage` first; branch on the first byte (`'"'` ⇒ empty list; `'{'` ⇒ decode as `{"style": [...]}`).

**Where:** `rest/styles/styles.go:85` (`json.RawMessage` decode tolerates the empty-string shape).

## 3. `LayerGroup.styles.style` is a mixed `[string|object]` array

**Symptom:** `GET /layergroups/{name}` for any layer group with default-styled members produces `"styles": {"style": ["", "", {...}, ""]}` — string entries (often empty) interspersed with style objects. The standard JSON decoder errors with `cannot unmarshal string into Go struct field LayerGroupStyles.style`.

**Workaround:** Custom `UnmarshalJSON` on `LayerGroupStyles` that decodes the inner `style` array via `[]json.RawMessage`, then per-element handles `'"'` (string) vs `'{'` (object) and produces `[]*Resource`. String entries are stored as `&Resource{Name: stringValue}` to preserve the `[]*Resource` field type.

**Where:** `rest/layergroups/types.go:108` (`Styles.UnmarshalJSON`); the same trick handles the mixed `Published` shape at `rest/layergroups/types.go:60`.

## 4. POST style endpoints need explicit `; charset=utf-8`

**Symptom:** Bare `Content-Type: application/json` 500s in some 2.x versions.

**Workaround:** Send `Content-Type: application/json; charset=utf-8`.

**Where:** Same site as quirk #1 — `rest/styles/styles.go:173` (the body Content-Type is set to `"application/json; charset=utf-8"`).

## 5. PostGIS publish requires the table to exist with attributes

**Symptom:** `POST /workspaces/{ws}/datastores/{ds}/featuretypes` returns `400 "no attributes"` if the named table is empty or doesn't exist.

**Workaround:** Tests bootstrap a real PostGIS table via `docker/postgis/init/01-lbldyt.sql` (creates `public.lbldyt(gid, name, label, geom)` with sample rows + GIST index). Production callers must ensure their target table exists before publishing it.

**Where:** `docker/postgis/init/01-lbldyt.sql`; tested in `rest/featuretypes/featuretypes_integration_test.go`.

## 6. `Settings.contact` returns the empty string when absent

**Symptom:** `GET /rest/settings` on a freshly initialized GeoServer returns `"contact": ""` (bare string) instead of an empty object. Decoding into a `*Contact` field with the standard JSON decoder fails with `cannot unmarshal string into Go struct field`.

**Workaround:** `*Contact` ships a custom `UnmarshalJSON` that treats both the empty-string and absent-field cases as a zero-value `Contact`, and decodes into the struct otherwise.

**Where:** `rest/settings/types.go` (`Contact.UnmarshalJSON`); regression-guarded by `rest/settings/settings_test.go:32` (`TestContact_UnmarshalEmptyString`).

## 7. Pagination drift across versions

**Symptom:** `GET /rest/layers` and `GET /rest/styles` paginate via `?startIndex=&count=` on GeoServer 2.18+ but return everything on older versions.

**Workaround:** Send pagination params and tolerate them being ignored on older servers. The client wraps this in `iter.Seq2[T, error]` with single-page fallback so callers iterate uniformly across versions.

**Where:** `rest/styles/styles.go:104` (`Client.Iter`), `rest/layers/layers.go:83` (`WorkspaceClient.Iter`).

## 8. URL building must escape per segment, not the whole path

**Symptom:** Workspace / layer names with spaces, slashes, or non-ASCII characters produce malformed URLs if a caller `fmt.Sprintf`s the path together. ACL rule strings carrying literal `*` wildcards are rejected by GeoServer's `StrictHttpFirewall` as "potentially malicious URL" if they end up double-encoded (`%25`).

**Workaround:** `transport.BuildURL(base, parts)` applies `url.PathEscape` to each segment before joining and preserves the encoding through `(*url.URL).String()` by setting `RawPath` alongside `Path`. Sub-clients reach it through `coreAdapter.URL(parts...)` (`geoserver.go:422`); never `fmt.Sprintf` REST paths.

**Where:** `internal/transport/url.go` (`BuildURL`); regression-guarded by `internal/transport/url_test.go`.

## 9. Empty `wfs:FeatureType` lists in capabilities

**Symptom:** Older GeoServer versions emit `<FeatureTypeList></FeatureTypeList>` (empty) while newer ones omit the element entirely.

**Workaround:** The WFS capabilities XML decoder treats both as an empty `FeatureTypeList`; no caller-visible difference. The WMS side does the same for empty `Layer` lists.

**Where:** `ows/wfs/types.go:106` (`FeatureTypeList`); `ows/wms/wms.go:114` (`ParseCapabilities`) returns `(*Capabilities, error)`.

## 10. Security service response keys differ across versions

**Symptom:** `GET /rest/security/roles` returns `{"roleNames": [...]}` on GeoServer pre-2.28 and `{"roles": [...]}` on 2.28+. Same drift on `/rest/security/usergroup/.../groups`.

**Workaround:** `GetRoles`, `GetUserRoles`, and `GetGroups` decode both shapes; whichever key has content wins.

**Where:** `rest/security/security.go:246` (`GetRoles` decode), `rest/security/security.go:292` (`GetUserRoles` and `GetGroups` reuse the same shape-tolerant pattern); types live in `rest/security/types.go:64`.

---

## When to update this catalog

- A new quirk surfaces during integration testing → add an entry here AND a regression test.
- A version drops out of the supported matrix (e.g., 2.27 LTS retires) → trim version-specific quirks for that version.
- A workaround is removed (e.g., GeoServer fixes the upstream bug and we drop the version that needed it) → mark as "Resolved in N.M+ — workaround removed in v1.x.y" rather than deleting.
