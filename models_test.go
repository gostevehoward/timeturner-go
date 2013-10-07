package timeturner

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"testing"
	"time"
)

var now time.Time = time.Date(2013, 10, 6, 0, 0, 0, 0, time.Local)

func setUp() *Database {
	connection, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	return InitializeDatabase(connection, func() time.Time { return now })
}

func addTimestampTestData(database *Database) {
	secondTime := now.Add(1 * time.Hour)
	thirdTime := now.Add(23 * time.Hour)
	for _, timestamp := range []time.Time{now, secondTime, thirdTime} {
		database.AddSnapshot(timestamp, "host1", "processes", "hello world!")
	}
}

func TestGetAllDays(t *testing.T) {
	database := setUp()
	addTimestampTestData(database)

	days := database.GetAllDays()
	expected := []string{"2013-10-06", "2013-10-07"}
	if len(days) != 2 {
		t.Fatalf("Unexpected days: %q", days)
	}
	for index, day := range days {
		if day.Format("2006-01-02") != expected[index] {
			t.Fatalf("Unexpected day at %d: %q", index, days)
		}
	}
}

func TestGetTimestamps(t *testing.T) {
	database := setUp()
	addTimestampTestData(database)

	timestamps := database.GetTimestamps(now)
	expected := []string{"00:00:00", "01:00:00"}
	if len(timestamps) != 2 {
		t.Fatalf("Unexpected timestamps: %q", timestamps)
	}
	for index, timestamp := range timestamps {
		if timestamp.Format("03:04:05") != expected[index] {
			t.Fatalf("Unexpected timestamp at %d: %q", index, timestamps)
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
		database.AddSnapshot(snapshot.Timestamp(), snapshot.Hostname, snapshot.Title, "")
	}

	snapshots := database.GetSnapshots(now)
	if len(snapshots) != 3 {
		t.Fatalf("Unexpected snapshots: %q", snapshots)
	}
	for index, snapshot := range snapshots {
		if snapshot.Hostname != data[index].Hostname || snapshot.Title != data[index].Title {
			t.Fatalf("Unexpected snapshot %q (expected %q): %q", snapshot, data[index], snapshots)
		}
	}
}

func TestGetSnapshotContents(t *testing.T) {
	database := setUp()

	expectedContents := "Hello world!"
	database.AddSnapshot(now, "host1", "processes", "other contents")
	database.AddSnapshot(now, "host1", "queries", expectedContents)

	contents := database.GetSnapshotContents(now, "host1", "queries")
	if contents != expectedContents {
		t.Fatalf("Unexpected contents: %q", contents)
	}
}

func TestCleanOldSnapshots(t *testing.T) {
	database := setUp()

	database.AddSnapshot(now, "host1", "processes", "")
	now = now.Add(100 * 24 * time.Hour)
	database.AddSnapshot(now, "host2", "queries", "")

	days := database.GetAllDays()
	if len(days) != 1 {
		t.Fatalf("Expected just one day: %q", days)
	}
}
