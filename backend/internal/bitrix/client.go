package bitrix

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

type CallError struct {
	Code   string
	Public string
}

func (e *CallError) Error() string {
	if e == nil {
		return "bitrix request failed"
	}
	if e.Code != "" {
		return e.Code
	}
	return e.Public
}

func PublicError(err error) string {
	var call *CallError
	if errors.As(err, &call) && call.Public != "" {
		return call.Public
	}
	return "Битрикс24 не ответил или отклонил запрос. Проверьте права вебхука."
}

type apiEnvelope struct {
	Result           json.RawMessage `json:"result"`
	Next             json.RawMessage `json:"next"`
	Error            json.RawMessage `json:"error"`
	ErrorDescription string          `json:"error_description"`
}

func (c *Client) Call(ctx context.Context, base *url.URL, method string, params map[string]any) (json.RawMessage, int, error) {
	endpoint, err := url.Parse(bitrixurl.MethodURL(base, method))
	if err != nil {
		return nil, 0, &CallError{Public: "Некорректный URL вебхука"}
	}
	raw, next, err := c.doGET(ctx, endpoint, params)
	if err != nil && isRetryableTransport(err) {
		return c.doJSON(ctx, endpoint, params)
	}
	return raw, next, err
}

func isRetryableTransport(err error) bool {
	var call *CallError
	if !errors.As(err, &call) {
		return false
	}
	return call.Code == "not_json" || call.Code == "unreachable"
}

func (c *Client) doJSON(ctx context.Context, endpoint *url.URL, params map[string]any) (json.RawMessage, int, error) {
	if params == nil {
		params = map[string]any{}
	}
	payload, err := json.Marshal(params)
	if err != nil {
		return nil, 0, &CallError{Public: "Не удалось подготовить запрос к Битрикс24"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, 0, &CallError{Public: "Не удалось подготовить запрос к Битрикс24"}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "RigIntel/1.0")
	return c.finish(req)
}

func (c *Client) doGET(ctx context.Context, endpoint *url.URL, params map[string]any) (json.RawMessage, int, error) {
	u := *endpoint
	q := u.Query()
	for k, v := range params {
		q.Set(k, fmt.Sprint(v))
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, &CallError{Public: "Не удалось подготовить запрос к Битрикс24"}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "RigIntel/1.0")
	return c.finish(req)
}

func (c *Client) finish(req *http.Request) (json.RawMessage, int, error) {
	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, &CallError{Code: "unreachable", Public: "Битрикс24 недоступен с сервера RigIntel"}
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBody+1))
	if err != nil {
		return nil, 0, &CallError{Code: "not_json", Public: "Битрикс24 вернул неполный ответ"}
	}
	if len(body) > maxBody {
		return nil, 0, &CallError{Code: "not_json", Public: "Ответ Битрикс24 слишком большой"}
	}
	env, err := parseEnvelope(body)
	if err != nil {
		return nil, 0, err
	}
	if code := errorCode(env.Error); code != "" || res.StatusCode >= 400 {
		if code == "" {
			code = strconv.Itoa(res.StatusCode)
		}
		return nil, 0, &CallError{Code: code, Public: publicBitrixError(code, env.ErrorDescription)}
	}
	next := 0
	if len(env.Next) > 0 && string(env.Next) != "null" {
		next, _ = strconv.Atoi(strings.Trim(string(env.Next), `"`))
	}
	return env.Result, next, nil
}

func parseEnvelope(body []byte) (apiEnvelope, error) {
	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return env, &CallError{Code: "not_json", Public: "Битрикс24 вернул не JSON. Проверьте URL вебхука."}
	}
	return env, nil
}

func errorCode(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	s := strings.TrimSpace(string(raw))
	if s == "null" || s == `""` || s == "false" || s == "0" {
		return ""
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString)
	}
	return ""
}

func publicBitrixError(code, description string) string {
	switch strings.ToLower(code) {
	case "insufficient_scope", "error_method_not_found", "access_denied", "invalid_credentials":
		return "У вебхука нет права «Структура компании» (department). Откройте вебхук в Битрикс, включите это право, сохраните и синхронизируйте снова."
	case "no_auth_found":
		return "Битрикс не принял вебхук. Скопируйте URL ещё раз после сохранения прав."
	case "query_limit_exceeded":
		return "Битрикс временно ограничил частоту запросов. Повторите синхронизацию через минуту."
	}
	if description != "" && len(description) <= 180 && !looksSecret(description) {
		return "Битрикс24 отклонил запрос: " + description
	}
	return "Битрикс24 отклонил запрос. Проверьте права вебхука: Пользователи и Структура компании."
}

func looksSecret(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "rest/") || strings.Contains(lower, "http")
}

func (c *Client) CurrentUser(ctx context.Context, base *url.URL) error {
	_, _, err := c.Call(ctx, base, "user.current", nil)
	return err
}

type Department struct {
	ID       int64
	ParentID *int64
	HeadID   *int64
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
		params := map[string]any{}
		if start > 0 {
			params["start"] = start
		}
		raw, next, err := c.Call(ctx, base, "department.get", params)
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
			head := asInt64(first(row, "UF_HEAD", "uf_head"))
			var headID *int64
			if head > 0 {
				headID = &head
			}
			out = append(out, Department{
				ID:       id,
				ParentID: parentID,
				HeadID:   headID,
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
		params := map[string]any{}
		if start > 0 {
			params["start"] = start
		}
		raw, next, err := c.Call(ctx, base, "user.get", params)
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
