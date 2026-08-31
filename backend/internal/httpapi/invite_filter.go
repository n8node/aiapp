package httpapi

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/n8node/aiapp/internal/repository"
)

func moscowLocation() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}

func parseDayStart(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", raw, moscowLocation())
	if err != nil {
		return nil, fmt.Errorf("invalid date")
	}
	return &t, nil
}

func parseDayEnd(raw string) (*time.Time, error) {
	start, err := parseDayStart(raw)
	if err != nil || start == nil {
		return start, err
	}
	end := start.Add(24 * time.Hour)
	return &end, nil
}

func parseInviteListFilter(q url.Values) (repository.InviteListFilter, error) {
	f := repository.InviteListFilter{
		Status:    strings.TrimSpace(q.Get("status")),
		UsedEmail: strings.TrimSpace(q.Get("used_email")),
	}
	var err error
	if f.CreatedFrom, err = parseDayStart(q.Get("created_from")); err != nil {
		return f, err
	}
	if f.CreatedTo, err = parseDayEnd(q.Get("created_to")); err != nil {
		return f, err
	}
	if f.UsedFrom, err = parseDayStart(q.Get("used_from")); err != nil {
		return f, err
	}
	if f.UsedTo, err = parseDayEnd(q.Get("used_to")); err != nil {
		return f, err
	}
	if v := strings.TrimSpace(q.Get("limit")); v != "" {
		n, convErr := strconv.Atoi(v)
		if convErr != nil || n < 0 {
			return f, fmt.Errorf("invalid limit")
		}
		f.Limit = n
	}
	if v := strings.TrimSpace(q.Get("offset")); v != "" {
		n, convErr := strconv.Atoi(v)
		if convErr != nil || n < 0 {
			return f, fmt.Errorf("invalid offset")
		}
		f.Offset = n
	}
	return f, nil
}
