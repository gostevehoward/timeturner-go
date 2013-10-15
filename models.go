package timeturner

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"github.com/coopernurse/gorp"
	"log"
	"os"
	"time"
)

func parseCsv(csvContents string) [][]string {
	reader := csv.NewReader(bytes.NewBufferString(csvContents))
	contents, err := reader.ReadAll()
	if err != nil {
		panic(err)
	} else {
		return contents
	}
}

func dumpCsv(contents [][]string) string {
	var csvContentsBuffer bytes.Buffer
	err := csv.NewWriter(&csvContentsBuffer).WriteAll(contents)
	if err != nil {
		panic(err)
	}
	return csvContentsBuffer.String()
}

const SCHEMA = `
CREATE TABLE IF NOT EXISTS Snapshot (
    Id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    UnixTimestamp INTEGER NOT NULL,
    Hostname VARCHAR(255) NOT NULL,
    Title VARCHAR(255) NOT NULL,
    CsvContents TEXT NOT NULL
);
`

type Snapshot struct {
	Id            int64
	UnixTimestamp int64
	Hostname      string
	Title         string
	CsvContents   string
}

func (snapshot Snapshot) Timestamp() time.Time {
	return time.Unix(snapshot.UnixTimestamp, 0)
}

func (snapshot Snapshot) Contents() [][]string {
	return parseCsv(snapshot.CsvContents)
}

type TimeturnerDatabase struct {
	mapper  gorp.DbMap
	nowFunc func() time.Time
}

func InitializeDatabase(connection *sql.DB, nowFunc func() time.Time, enableLogging bool,
) *TimeturnerDatabase {
	mapper := gorp.DbMap{Db: connection, Dialect: gorp.SqliteDialect{}}
	if enableLogging {
		mapper.TraceOn("[gorp]", log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile))
	}
	mapper.AddTable(Snapshot{}).SetKeys(true, "Id")

	_, err := mapper.Exec(SCHEMA)
	if err != nil {
		panic(err)
	}

	return &TimeturnerDatabase{mapper, nowFunc}
}

func (database *TimeturnerDatabase) cleanOldSnapshots() {
	oldestAllowedTimestamp := database.nowFunc().AddDate(0, 0, -14)
	query := "DELETE FROM Snapshot WHERE UnixTimestamp < ?"
	_, err := database.mapper.Exec(query, oldestAllowedTimestamp.Unix())
	if err != nil {
		panic(err)
	}
}

func (database *TimeturnerDatabase) AddSnapshot(timestamp time.Time, hostname string, title string,
	contents [][]string) {
	csvContents := dumpCsv(contents)

	snapshot, alreadyExists := database.GetSnapshotWithContents(timestamp, hostname, title)
	if alreadyExists {
		snapshot.CsvContents = csvContents
		numUpdated, err := database.mapper.Update(&snapshot)
		if err != nil {
			panic(err)
		}
		if numUpdated != 1 {
			panic(
				fmt.Sprintf(
					"Updated %d rows overwriting snapshot: timestamp=%v, hostname=%v, title=%v",
					timestamp, hostname, title,
				),
			)
		}
	} else {
		snapshot := &Snapshot{-1, timestamp.Unix(), hostname, title, csvContents}
		err := database.mapper.Insert(snapshot)
		if err != nil {
			panic(err)
		}
		database.cleanOldSnapshots()
	}
}

func (database *TimeturnerDatabase) querySnapshots(query string, args ...interface{}) []Snapshot {
	var rows []Snapshot
	_, err := database.mapper.Select(&rows, query, args...)
	if err != nil {
		panic(err)
	}
	return rows
}

func uniqueTimestamps(snapshots []Snapshot, mapTimestamp func(time.Time) time.Time) []time.Time {
	timestamps := make([]time.Time, 0)
	seenMap := make(map[time.Time]bool)
	for _, snapshot := range snapshots {
		timestamp := mapTimestamp(snapshot.Timestamp())
		if _, seen := seenMap[timestamp]; !seen {
			timestamps = append(timestamps, timestamp)
			seenMap[timestamp] = true
		}
	}
	return timestamps
}

func (database *TimeturnerDatabase) GetAllDays() []time.Time {
	query := "SELECT DISTINCT UnixTimestamp FROM Snapshot ORDER BY UnixTimestamp"
	rows := database.querySnapshots(query)
	return uniqueTimestamps(rows, func(timestamp time.Time) time.Time {
		year, month, day := timestamp.Date()
		return time.Date(year, month, day, 0, 0, 0, 0, timestamp.Location())
	})
}

func (database *TimeturnerDatabase) GetTimestamps(day time.Time) []time.Time {
	query := "SELECT DISTINCT UnixTimestamp FROM Snapshot " +
		"WHERE UnixTimestamp >= ? AND UnixTimestamp < ? ORDER BY UnixTimestamp"
	rows := database.querySnapshots(query, day.Unix(), day.AddDate(0, 0, 1).Unix())
	return uniqueTimestamps(rows, func(timestamp time.Time) time.Time { return timestamp })
}

func (database *TimeturnerDatabase) GetSnapshots(timestamp time.Time) []Snapshot {
	query := "SELECT Id, UnixTimestamp, Hostname, Title FROM Snapshot WHERE UnixTimestamp = ? " +
		"ORDER BY Hostname, Title"
	rows := database.querySnapshots(query, timestamp.Unix())

	return rows
}

func (database *TimeturnerDatabase) GetSnapshotWithContents(timestamp time.Time, hostname string,
	title string) (snapshot Snapshot, ok bool) {
	query := "SELECT * FROM Snapshot WHERE UnixTimestamp = ? AND Hostname = ? AND Title = ?"
	rows := database.querySnapshots(query, timestamp.Unix(), hostname, title)
	if len(rows) == 0 {
		return Snapshot{}, false
	} else if len(rows) == 1 {
		return rows[0], true
	} else {
		panic(
			fmt.Sprintf("Multiple snapshots found: timestamp %v, hostname %v, title %v",
				timestamp, hostname, title,
			),
		)
	}
}
