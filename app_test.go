package timeturner

import (
	"testing"
	"time"
)

func assertTimestampResults(t *testing.T, expected time.Time, timestamp time.Time, err error) {
	if err != nil {
		t.Fatalf("Got error for date: %v", err)
	}
	if !timestamp.Equal(expected) {
		t.Fatalf("Expected %v, got %v", expected, timestamp)
	}
}

func TestParseTimestampWithDate(t *testing.T) {
	timestamp, err := parseTimestamp(map[string]string{"date": "2013-10-05"})
	expected := time.Date(2013, 10, 5, 0, 0, 0, 0, time.Local)
	assertTimestampResults(t, expected, timestamp, err)
}

func TestParseTimestampWithDateAndTime(t *testing.T) {
	timestamp, err := parseTimestamp(map[string]string{"date": "2013-10-05", "time": "15:32:44"})
	expected := time.Date(2013, 10, 5, 15, 32, 44, 0, time.Local)
	assertTimestampResults(t, expected, timestamp, err)
}

func TestParseTimestampWithInvalidDate(t *testing.T) {
	_, err := parseTimestamp(map[string]string{"date": "2013-10-43"})
	if err == nil {
		t.Fatalf("No error for invalid date")
	}
}
