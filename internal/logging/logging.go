// SPDX-License-Identifier: AGPL-3.0-or-later

// Package logging builds the log handler of the server.
package logging

import (
	"io"
	"log/slog"
	"time"
)

// timeFormat is the format of the time of each log line: YYYY-MM-DD HH:MM:SS.
const timeFormat = "2006-01-02 15:04:05"

// NewHandler returns a slog text handler that writes to w. It shows the time
// of each line in loc. Before the server loads the time zone, loc is time.UTC.
func NewHandler(w io.Writer, loc *time.Location) *slog.TextHandler {
	return slog.NewTextHandler(w, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			// slog also passes other attributes named "time". Value.Time panics on
			// a value that is not a time, and a log call must never crash the server.
			if len(groups) == 0 && attr.Key == slog.TimeKey && attr.Value.Kind() == slog.KindTime {
				return slog.String(slog.TimeKey, attr.Value.Time().In(loc).Format(timeFormat))
			}
			return attr
		},
	})
}
