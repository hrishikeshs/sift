package parse

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ParseSince(s string) (time.Time, error) {
	now := time.Now().UTC()

	switch strings.ToLower(s) {
	case "today":
		y, m, d := now.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), nil
	case "yesterday":
		y, m, d := now.AddDate(0, 0, -1).Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), nil
	}

	// Try ISO date: 2026-04-20
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	// Try ISO datetime: 2026-04-20T15:00:00Z
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try duration: 2w, 3d, 1h, 30m
	if len(s) >= 2 {
		numStr := s[:len(s)-1]
		unit := s[len(s)-1]
		n, err := strconv.Atoi(numStr)
		if err == nil {
			switch unit {
			case 'h':
				return now.Add(-time.Duration(n) * time.Hour), nil
			case 'd':
				return now.AddDate(0, 0, -n), nil
			case 'w':
				return now.AddDate(0, 0, -n*7), nil
			case 'M':
				return now.AddDate(0, -n, 0), nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized time format: %q (use 2w, 3d, 1h, yesterday, today, or YYYY-MM-DD)", s)
}
