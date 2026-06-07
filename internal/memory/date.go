package memory

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// dateLayout is the on-disk format for all recall dates: YYYY-MM-DD.
const dateLayout = "2006-01-02"

// Date is a calendar date (no time-of-day) that marshals to/from YYYY-MM-DD in
// YAML frontmatter. The zero value represents "no date" and is omitted on emit.
type Date struct {
	time.Time
}

// Today returns the current date in local time.
func Today() Date {
	now := time.Now()
	return Date{time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)}
}

// ParseDate parses a YYYY-MM-DD string into a Date.
func ParseDate(s string) (Date, error) {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q (want YYYY-MM-DD): %w", s, err)
	}
	return Date{t}, nil
}

// String renders the date as YYYY-MM-DD, or "" when zero.
func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return d.Time.Format(dateLayout)
}

// MarshalYAML emits the date as a YYYY-MM-DD string, or null when zero.
func (d Date) MarshalYAML() (any, error) {
	if d.IsZero() {
		return nil, nil
	}
	return d.Time.Format(dateLayout), nil
}

// UnmarshalYAML parses a YYYY-MM-DD scalar. Empty/null leaves the date zero.
func (d *Date) UnmarshalYAML(value *yaml.Node) error {
	if value.Value == "" || value.Tag == "!!null" {
		return nil
	}
	parsed, err := ParseDate(value.Value)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
