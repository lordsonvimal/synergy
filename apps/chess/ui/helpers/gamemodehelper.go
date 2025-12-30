package helpers

import "fmt"

func FormatTime(ns int64) string {
	mins := ns / 1_000_000_000 / 60
	return fmt.Sprintf("%dm", mins)
}

func FormatInc(ns int64) string {
	secs := ns / 1_000_000_000
	return fmt.Sprintf("%ds", secs)
}
