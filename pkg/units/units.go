// Package units parses the scalar literals used in scenario attributes.
//
// The timeline works exclusively in integer milliseconds, so every duration is
// normalised here at the boundary rather than carried around as a float.
package units

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ParseMillis converts a duration literal such as "600ms", "1.2s" or "750" to
// whole milliseconds. A bare number is read as milliseconds.
func ParseMillis(s string) (int, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return 0, fmt.Errorf("empty duration")
	}

	num := raw
	mult := 1.0
	switch {
	case strings.HasSuffix(raw, "ms"):
		num = strings.TrimSuffix(raw, "ms")
	case strings.HasSuffix(raw, "s"):
		num = strings.TrimSuffix(raw, "s")
		mult = 1000
	case strings.HasSuffix(raw, "m"):
		num = strings.TrimSuffix(raw, "m")
		mult = 60000
	}

	v, err := strconv.ParseFloat(strings.TrimSpace(num), 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a duration", s)
	}
	if v < 0 {
		return 0, fmt.Errorf("duration %q is negative", s)
	}
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return 0, fmt.Errorf("%q is not a duration", s)
	}
	return int(math.Round(v * mult)), nil
}

// ParseFloat reads a plain number such as a speed multiplier.
func ParseFloat(s string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a number", s)
	}
	return v, nil
}

// ParseBool reads true/false/yes/no/on/off.
func ParseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "yes", "on", "1":
		return true, nil
	case "false", "no", "off", "0":
		return false, nil
	}
	return false, fmt.Errorf("%q is not a boolean", s)
}
