// Package monitor is the v2 sub-client for the GeoServer
// /rest/monitor/requests endpoint — read-only audit log of OWS and
// REST requests handled by the server. Exposed by the gs-monitor
// extension; absent installs return ErrNotFound.
//
// The endpoint serves CSV / Excel / ZIP / HTML — there is no JSON
// representation. The SDK fetches the CSV form and decodes into
// typed [Request] structs. For raw access (e.g. to drop into a
// reporting pipeline) use [Client.ListRaw].
package monitor

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
type Core interface {
	URL(parts ...string) (string, error)
	DoStream(ctx context.Context, op string, method, requestURL string, query map[string]string) (io.ReadCloser, int, error)
}

// Client is the v2 monitor sub-client.
type Client struct {
	core Core
}

// New constructs the monitor sub-client.
func New(core Core) *Client {
	return &Client{core: core}
}

// ListOptions filters the audit log returned by [Client.List] /
// [Client.ListRaw]. All fields are optional.
type ListOptions struct {
	// From / To bound the request StartTime. ISO 8601 strings, any
	// precision (e.g. "2026-04-23", "2026-04-23T16:16:44").
	From, To string

	// Filter is the upstream `attributeName:OP:value` shape — see
	// the GeoServer monitor REST docs. OP is one of EQ, NEQ, LT, LTE,
	// GT, GTE, IN.
	Filter string

	// Order is "attributeName[;ASC|DESC]"; default is server-defined
	// (usually descending start time).
	Order string

	// Offset and Count paginate the result set.
	Offset, Count int

	// Live, when set, restricts to live (Running / Pending /
	// Cancelling) requests if true, completed (Finished / Failed) if
	// false. Leave the pointer nil to return both.
	Live *bool

	// Fields restricts the returned columns. Empty means "all fields".
	// CSV column-name set documented in [Request].
	//
	// GeoServer 2.27 / 2.28 quirk: the upstream API doc says this is
	// a comma-separated list, but the server-side parser only accepts
	// a single property name — passing two or more comma-joined names
	// returns 500 "No such property 'Foo,Bar' for object Request".
	// Until that is fixed upstream, supply at most one entry; for
	// multiple-column projections, fetch all fields and discard
	// client-side, or use [Client.ListRaw] and decode the CSV yourself.
	Fields []string
}

// Request is one audit-log entry. The struct mirrors the documented
// CSV column set; missing or empty CSV cells decode to the zero
// value for the field.
type Request struct {
	ID                  int64
	Path                string
	Service             string
	Operation           string
	SubOperation        string
	OWSVersion          string
	HTTPMethod          string
	QueryString         string
	Category            string
	StartTime           time.Time
	EndTime             time.Time
	TotalTime           int64 // milliseconds
	Status              string
	ResponseStatus      int
	ResponseLength      int64
	ResponseContentType string
	ErrorMessage        string
	Host                string
	InternalHost        string
	RemoteAddr          string
	RemoteHost          string
	RemoteUser          string
	RemoteUserAgent     string
	RemoteCity          string
	RemoteCountry       string
	RemoteLat           float64
	RemoteLon           float64
	Bbox                string
	Resources           []string
	BodyContentLength   int64
	BodyContentType     string
	BodyAsString        string
	HTTPReferer         string
	CacheResult         string
	MissReason          string
}

// List fetches the audit log and decodes it into typed [Request]
// values. For very large result sets prefer [Client.ListRaw] and
// stream-decode the CSV directly.
func (c *Client) List(ctx context.Context, opts ListOptions) ([]Request, error) {
	const op = "Monitor.List"
	rc, err := c.ListRaw(ctx, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	rows, err := decodeCSV(rc)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return rows, nil
}

// ListRaw returns the raw CSV body. The caller MUST close the
// returned [io.ReadCloser]. Use this for streaming pipelines or
// when the typed [Request] subset doesn't cover every column you
// need (the GeoServer schema includes ~35 fields; the SDK promotes
// the well-known subset, callers drop into the raw stream for
// extras like LabellingProcessingTime / ResourcesProcessingTime).
func (c *Client) ListRaw(ctx context.Context, opts ListOptions) (io.ReadCloser, error) {
	const op = "Monitor.ListRaw"
	u, err := c.core.URL("rest", "monitor", "requests.csv")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	q := opts.query()
	rc, _, err := c.core.DoStream(ctx, op, http.MethodGet, u, q)
	if err != nil {
		return nil, err
	}
	return rc, nil
}

// Get fetches a single audit-log entry by ID. Returns a *APIError
// wrapping ErrNotFound if the entry doesn't exist.
func (c *Client) Get(ctx context.Context, id int64) (*Request, error) {
	const op = "Monitor.Get"
	if id <= 0 {
		return nil, errors.New(op + ": id must be positive")
	}
	u, err := c.core.URL("rest", "monitor", "requests", strconv.FormatInt(id, 10)+".csv")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	rc, _, err := c.core.DoStream(ctx, op, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	rows, err := decodeCSV(rc)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%s: empty response", op)
	}
	return &rows[0], nil
}

func (o ListOptions) query() map[string]string {
	q := map[string]string{}
	if o.From != "" {
		q["from"] = o.From
	}
	if o.To != "" {
		q["to"] = o.To
	}
	if o.Filter != "" {
		q["filter"] = o.Filter
	}
	if o.Order != "" {
		q["order"] = o.Order
	}
	if o.Offset > 0 {
		q["offset"] = strconv.Itoa(o.Offset)
	}
	if o.Count > 0 {
		q["count"] = strconv.Itoa(o.Count)
	}
	if o.Live != nil {
		q["live"] = strconv.FormatBool(*o.Live)
	}
	if len(o.Fields) > 0 {
		q["fields"] = strings.Join(o.Fields, ",")
	}
	return q
}

// decodeCSV turns the CSV body into typed Request values. The first
// row is the header; subsequent rows map by column name into the
// struct fields documented in [Request]. Unknown columns are
// silently dropped.
func decodeCSV(r io.Reader) ([]Request, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1 // tolerate ragged rows
	header, err := cr.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, fmt.Errorf("read header: %w", err)
	}
	out := make([]Request, 0)
	for {
		row, err := cr.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return out, nil
			}
			return nil, fmt.Errorf("read row: %w", err)
		}
		var req Request
		for i, col := range header {
			if i >= len(row) {
				break
			}
			req.set(col, row[i])
		}
		out = append(out, req)
	}
}

// set decodes one CSV column into the matching struct field.
func (r *Request) set(col, val string) {
	if val == "" {
		return
	}
	switch col {
	case "Id":
		r.ID, _ = strconv.ParseInt(val, 10, 64)
	case "Path":
		r.Path = val
	case "Service":
		r.Service = val
	case "Operation":
		r.Operation = val
	case "SubOperation":
		r.SubOperation = val
	case "OwsVersion":
		r.OWSVersion = val
	case "HttpMethod":
		r.HTTPMethod = val
	case "QueryString":
		r.QueryString = val
	case "Category":
		r.Category = val
	case "StartTime":
		r.StartTime, _ = time.Parse("2006-01-02T15:04:05.000", val)
	case "EndTime":
		r.EndTime, _ = time.Parse("2006-01-02T15:04:05.000", val)
	case "TotalTime":
		r.TotalTime, _ = strconv.ParseInt(val, 10, 64)
	case "Status":
		r.Status = val
	case "ResponseStatus":
		n, _ := strconv.Atoi(val)
		r.ResponseStatus = n
	case "ResponseLength":
		r.ResponseLength, _ = strconv.ParseInt(val, 10, 64)
	case "ResponseContentType":
		r.ResponseContentType = val
	case "ErrorMessage":
		r.ErrorMessage = val
	case "Host":
		r.Host = val
	case "InternalHost":
		r.InternalHost = val
	case "RemoteAddr":
		r.RemoteAddr = val
	case "RemoteHost":
		r.RemoteHost = val
	case "RemoteUser":
		r.RemoteUser = val
	case "RemoteUserAgent":
		r.RemoteUserAgent = val
	case "RemoteCity":
		r.RemoteCity = val
	case "RemoteCountry":
		r.RemoteCountry = val
	case "RemoteLat":
		r.RemoteLat, _ = strconv.ParseFloat(val, 64)
	case "RemoteLon":
		r.RemoteLon, _ = strconv.ParseFloat(val, 64)
	case "Bbox":
		r.Bbox = val
	case "Resources", "ResourcesList":
		// Resources is "[a, b]"; ResourcesList is "a, b". Either way,
		// split on comma after stripping the optional brackets.
		s := strings.TrimSpace(val)
		s = strings.TrimPrefix(s, "[")
		s = strings.TrimSuffix(s, "]")
		parts := strings.Split(s, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				r.Resources = append(r.Resources, p)
			}
		}
	case "BodyContentLength":
		r.BodyContentLength, _ = strconv.ParseInt(val, 10, 64)
	case "BodyContentType":
		r.BodyContentType = val
	case "BodyAsString":
		r.BodyAsString = val
	case "HttpReferer":
		r.HTTPReferer = val
	case "CacheResult":
		r.CacheResult = val
	case "MissReason":
		r.MissReason = val
	}
}
