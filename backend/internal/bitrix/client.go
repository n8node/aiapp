package bitrix

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/n8node/aiapp/internal/bitrixurl"
)

const maxBody = 2 << 20

type Client struct {
	http *http.Client
}

func NewClient() *Client {
	dialer := &net.Dialer{Timeout: 8 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("resolve host")
			}
			var last error
			for _, ipa := range ips {
				if bitrixurl.BlockedIP(ipa.IP) {
					last = fmt.Errorf("blocked host")
					continue
				}
				conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ipa.IP.String(), port))
				if err != nil {
					last = err
					continue
				}
				return conn, nil
			}
			if last == nil {
				last = fmt.Errorf("blocked host")
			}
			return nil, last
		},
		DisableKeepAlives: true,
	}
	return &Client{
		http: &http.Client{
			Timeout:   20 * time.Second,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 2 {
					return fmt.Errorf("too many redirects")
				}
				if !strings.EqualFold(req.URL.Scheme, "https") {
					return fmt.Errorf("https required")
				}
				if err := bitrixurl.AssertPublicHost(req.URL.Hostname()); err != nil {
					return err
				}
				return nil
			},
		},
	}
}

type apiEnvelope struct {
	Result json.RawMessage `json:"result"`
	Next   json.RawMessage `json:"next"`
	Error  string          `json:"error"`
}

func (c *Client) Call(ctx context.Context, base *url.URL, method string, query url.Values) (json.RawMessage, int, error) {
	endpoint, err := url.Parse(bitrixurl.MethodURL(base, method))
	if err != nil {
		return nil, 0, fmt.Errorf("bitrix url")
	}
	if query != nil {
		endpoint.RawQuery = query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("bitrix request")
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("bitrix unreachable")
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBody+1))
	if err != nil {
		return nil, 0, fmt.Errorf("bitrix response")
	}
	if len(body) > maxBody {
		return nil, 0, fmt.Errorf("bitrix response")
	}
	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, 0, fmt.Errorf("bitrix response")
	}
	if env.Error != "" || res.StatusCode >= 400 {
		return nil, 0, fmt.Errorf("bitrix rejected")
	}
	next := 0
	if len(env.Next) > 0 && string(env.Next) != "null" {
		next, _ = strconv.Atoi(strings.Trim(string(env.Next), `"`))
	}
	return env.Result, next, nil
}

func (c *Client) CurrentUser(ctx context.Context, base *url.URL) error {
	_, _, err := c.Call(ctx, base, "user.current", nil)
	return err
}

type Department struct {
	ID       int64
	ParentID *int64
	Name     string
	Sort     int
}

type User struct {
	ID            int64
	Email         string
	Name          string
	LastName      string
	Active        bool
	DepartmentIDs []int64
}

func (c *Client) ListDepartments(ctx context.Context, base *url.URL) ([]Department, error) {
	var out []Department
	start := 0
	for {
		q := url.Values{}
		if start > 0 {
			q.Set("start", strconv.Itoa(start))
		}
		raw, next, err := c.Call(ctx, base, "department.get", q)
		if err != nil {
			return nil, err
		}
		rows, err := parseObjects(raw)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			id := asInt64(row["ID"])
			if id == 0 {
				id = asInt64(row["id"])
			}
			if id == 0 {
				continue
			}
			parent := asInt64(row["PARENT"])
			var parentID *int64
			if parent > 0 {
				parentID = &parent
			}
			out = append(out, Department{
				ID:       id,
				ParentID: parentID,
				Name:     asString(first(row, "NAME", "name")),
				Sort:     int(asInt64(first(row, "SORT", "sort"))),
			})
		}
		if next == 0 {
			break
		}
		start = next
	}
	return out, nil
}

func (c *Client) ListUsers(ctx context.Context, base *url.URL) ([]User, error) {
	var out []User
	start := 0
	for {
		q := url.Values{}
		if start > 0 {
			q.Set("start", strconv.Itoa(start))
		}
		raw, next, err := c.Call(ctx, base, "user.get", q)
		if err != nil {
			return nil, err
		}
		rows, err := parseObjects(raw)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			id := asInt64(first(row, "ID", "id"))
			if id == 0 {
				continue
			}
			out = append(out, User{
				ID:            id,
				Email:         strings.ToLower(strings.TrimSpace(asString(first(row, "EMAIL", "email")))),
				Name:          asString(first(row, "NAME", "name")),
				LastName:      asString(first(row, "LAST_NAME", "last_name")),
				Active:        asBool(first(row, "ACTIVE", "active")),
				DepartmentIDs: asInt64Slice(first(row, "UF_DEPARTMENT", "uf_department")),
			})
		}
		if next == 0 {
			break
		}
		start = next
	}
	return out, nil
}

func parseObjects(raw json.RawMessage) ([]map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err == nil {
		return rows, nil
	}
	var one map[string]any
	if err := json.Unmarshal(raw, &one); err != nil {
		return nil, fmt.Errorf("bitrix response")
	}
	return []map[string]any{one}, nil
}

func first(row map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := row[k]; ok && v != nil {
			return v
		}
	}
	return nil
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case json.Number:
		return t.String()
	default:
		return ""
	}
}

func asInt64(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case json.Number:
		n, _ := t.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	case int:
		return int64(t)
	case int64:
		return t
	default:
		return 0
	}
}

func asBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "Y" || t == "1" || strings.EqualFold(t, "true")
	case float64:
		return t != 0
	default:
		return false
	}
}

func asInt64Slice(v any) []int64 {
	switch t := v.(type) {
	case []any:
		out := make([]int64, 0, len(t))
		for _, item := range t {
			if n := asInt64(item); n > 0 {
				out = append(out, n)
			}
		}
		return out
	default:
		if n := asInt64(v); n > 0 {
			return []int64{n}
		}
		return []int64{}
	}
}
