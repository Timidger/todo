package todo

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const AUDIT_LOG = "audit_log/"
const AUDIT_MINIMUM_FIELDS = 6 // XXX don't need "Notes"
const AUDIT_FIELDS = "Body_content, Due_date, Repeat, Overdue_days," +
	"Category, DateCompleted, Notes" +
	"\n"

type Records []Record

func (records Records) Len() int {
	return len(records)
}

func (records Records) Less(i, j int) bool {
	if records[i].DateCompleted.Equal(records[j].DateCompleted) {
		return true
	}
	return records[i].DateCompleted.Before(records[j].DateCompleted)
}
func (records Records) Swap(i, j int) {
	records[i], records[j] = records[j], records[i]
}

// NOTE This is an append-only struct.
// If new fields are added, put them at the end and update ALL functions.
type Record struct {
	Body_content string
	Due_date     time.Time
	Repeat       *time.Duration
	Overdue_days int
	// This is actually determined at load time again,
	// since audit logs store the category by virtue of being in
	// separate directories.
	Category      string
	DateCompleted time.Time
	Annotation    string
}

func (record Record) Marshal() []string {
	body_content := record.Body_content
	due_date := record.Due_date.Format(EXPLICIT_TIME_FORMAT)
	repeat := ""
	if record.Repeat != nil {
		repeat = record.Repeat.String()
	}
	overdue_days := strconv.Itoa(record.Overdue_days)
	// Category determined at load time, from directory of audit_log
	category := ""
	date_completed := record.DateCompleted.Format(RECORD_TIME_FORMAT)
	annotation := record.Annotation
	return []string{
		body_content,
		due_date,
		repeat,
		overdue_days,
		category,
		date_completed,
		annotation,
	}
}

func (record Record) String() string {
	completed := record.DateCompleted.Format(RECORD_TIME_FORMAT)
	overdue := ""
	date_due := record.Due_date.AddDate(0, 0, record.Overdue_days)
	if date_due.Before(record.DateCompleted) {
		overdue_days := int(math.Floor(record.DateCompleted.Sub(date_due).Hours() / 24))
		if overdue_days != 0 {
			overdue = fmt.Sprintf(RED+" (overdue %d days)"+RESET, overdue_days)
		}
	}

	category_name := ""
	if record.Category != "" {
		category_name = "(" + record.Category + ")"
	}

	trimmed_content := strings.TrimSuffix(record.Body_content, "\n")
	postamble := fmt.Sprintf("%-15s%s", category_name, overdue)
	audit_entry := HardWrapString(trimmed_content, 60,
		completed, len(completed)+2, postamble, " ")
	if record.Annotation != "" {
		if len(record.Annotation) >= 60 {
			audit_entry += HardWrapString(record.Annotation, 60,
				"\n                         ┃  ", len(completed)+5, " ", "┃  ")
		} else {
			audit_entry += "\n                         ┗━ " + record.Annotation
		}
	}
	return audit_entry
}

func Unmarshal(fields []string) Record {
	if len(fields) < AUDIT_MINIMUM_FIELDS {
		panic(fmt.Sprintf("actual %d != expected at least %d", len(fields), AUDIT_MINIMUM_FIELDS))
	}
	var record Record
	record.Body_content = fields[0]
	record.Due_date, _ = time.Parse(EXPLICIT_TIME_FORMAT, fields[1])
	if fields[2] != "" {
		repeat, _ := time.ParseDuration(fields[2])
		record.Repeat = &repeat
	}
	overdue_days, _ := strconv.ParseInt(fields[3], 10, 32)
	record.Overdue_days = int(overdue_days)
	record.Category = fields[4]
	record.DateCompleted, _ = time.Parse(RECORD_TIME_FORMAT, fields[5])

	if len(fields) >= 7 {
		record.Annotation = fields[6]
	}

	return record
}
