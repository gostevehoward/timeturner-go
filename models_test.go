package timeturner

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"testing"
	"time"
)

var now time.Time = time.Date(2013, 10, 6, 0, 0, 0, 0, time.Local)

func setUp() Database {
	connection, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	return InitializeDatabase(connection, func() time.Time { return now }, false)
}

func wrapSimpleContents(contents string) [][]string {
	return [][]string{
		{"column"},
		{contents},
	}
}

func addTimestampTestData(database Database) {
	secondTime := now.Add(1 * time.Hour)
	thirdTime := now.Add(24 * time.Hour)
	for _, timestamp := range []time.Time{now, secondTime, thirdTime} {
		database.AddSnapshot(timestamp, "host1", "processes", [][]string{})
	}
}

func TestGetAllDays(t *testing.T) {
	database := setUp()
	addTimestampTestData(database)

	days := database.GetAllDays()
	expected := []string{"2013-10-06", "2013-10-07"}
	if len(days) != 2 {
		t.Fatalf("Unexpected days: %v", days)
	}
	for index, day := range days {
		if day.Format("2006-01-02") != expected[index] {
			t.Fatalf("Unexpected day at %d: %v", index, days)
		}
	}
}

func TestGetTimestamps(t *testing.T) {
	database := setUp()
	addTimestampTestData(database)

	timestamps := database.GetTimestamps(now)
	expected := []string{"00:00:00", "01:00:00"}
	if len(timestamps) != 2 {
		t.Fatalf("Unexpected timestamps: %v", timestamps)
	}
	for index, timestamp := range timestamps {
		if timestamp.Format("15:04:05") != expected[index] {
			t.Fatalf("Unexpected timestamp at %d: %v", index, timestamps)
		}
		if timestamp.Location() != time.Local {
			t.Fatalf("Expected local timezone, got %b", timestamp.Location())
		}
	}
}

func TestGetSnapshots(t *testing.T) {
	database := setUp()

	data := []Snapshot{
		{-1, now.Unix(), "host1", "processes", ""},
		{-1, now.Unix(), "host1", "queries", ""},
		{-1, now.Unix(), "host2", "processes", ""},
		{-1, now.Add(time.Hour).Unix(), "host2", "queries", ""},
	}

	for _, snapshot := range data {
		database.AddSnapshot(snapshot.Timestamp(), snapshot.Hostname, snapshot.Title, [][]string{})
	}

	snapshots := database.GetSnapshots(now)
	if len(snapshots) != 3 {
		t.Fatalf("Unexpected snapshots: %v", snapshots)
	}
	for index, snapshot := range snapshots {
		if snapshot.Hostname != data[index].Hostname || snapshot.Title != data[index].Title {
			t.Fatalf("Unexpected snapshot %v (expected %v): %v", snapshot, data[index], snapshots)
		}
	}
}

func TestGetSnapshotWithContents(t *testing.T) {
	database := setUp()

	database.AddSnapshot(now, "host1", "processes", wrapSimpleContents("other data"))
	database.AddSnapshot(now, "host1", "queries", wrapSimpleContents("Hello world!"))

	snapshot, ok := database.GetSnapshotWithContents(now, "host1", "queries")
	if !ok {
		t.Fatalf("Failed to find snapshot contents")
	}
	expectedContents := "column\nHello world!\n"
	if snapshot.CsvContents != expectedContents {
		t.Fatalf("Unexpected contents: %v", snapshot.Contents)
	}

	_, ok = database.GetSnapshotWithContents(now, "host2", "foobar")
	if ok {
		t.Fatalf("Found contents for nonexistent snapshot")
	}
}

func TestCleanOldSnapshots(t *testing.T) {
	database := setUp()

	database.AddSnapshot(now, "host1", "processes", [][]string{})
	now = now.AddDate(0, 0, 100)
	database.AddSnapshot(now, "host2", "queries", [][]string{})

	days := database.GetAllDays()
	if len(days) != 1 {
		t.Fatalf("Expected just one day: %v", days)
	}
}

func TestOverwriteExistingSnapshot(t *testing.T) {
	database := setUp()

	database.AddSnapshot(now, "host1", "queries", wrapSimpleContents("hello world"))
	database.AddSnapshot(now, "host1", "queries", wrapSimpleContents("goodbye cruel world"))

	snapshots := database.GetSnapshots(now)
	if len(snapshots) != 1 {
		t.Fatalf("Expected only one snapshot, found %d", len(snapshots))
	}

	snapshot, _ := database.GetSnapshotWithContents(now, "host1", "queries")
	if snapshot.Contents()[1][0] != "goodbye cruel world" {
		t.Fatalf("Unexpected contents: %v", snapshot.Contents)
	}
}
