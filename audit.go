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
const AUDIT_FIELDS = "BodyContent, DueDate, Repeat, OverdueDays," +
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

type Record struct {
	BodyContent string
	DueDate     time.Time
	Repeat      *string
	OverdueDays int
	// This is actually determined at load time again,
	// since audit logs store the category by virtue of being in
	// separate directories.
	Category      string
	DateCompleted time.Time
	Annotation    string
}

func (record Record) Marshal() []string {
	bodyContent := record.BodyContent
	dueDate := record.DueDate.Format(EXPLICIT_TIME_FORMAT)
	repeat := ""
	if record.Repeat != nil {
		repeat = *record.Repeat
	}
	overdueDays := strconv.Itoa(record.OverdueDays)
	// Category determined at load time, from directory of audit_log
	category := ""
	dateCompleted := record.DateCompleted.Format(RECORD_TIME_FORMAT)
	annotation := record.Annotation
	return []string{
		bodyContent,
		dueDate,
		repeat,
		overdueDays,
		category,
		dateCompleted,
		annotation,
	}
}

func (record Record) String() string {
	completed := record.DateCompleted.Format(RECORD_TIME_FORMAT)
	overdue := ""
	dateDue := record.DueDate.AddDate(0, 0, record.OverdueDays)
	if dateDue.Before(record.DateCompleted) {
		overdueDays := int(math.Floor(record.DateCompleted.Sub(dateDue).Hours() / 24))
		if overdueDays != 0 {
			overdue = fmt.Sprintf(RED+" (overdue %d days)"+RESET, overdueDays)
		}
	}

	categoryName := ""
	if record.Category != "" {
		categoryName = "(" + record.Category + ")"
	}

	trimmedContent := strings.TrimSuffix(record.BodyContent, "\n")
	postamble := fmt.Sprintf("%-15s%s", categoryName, overdue)
	audit_entry := HardWrapString(trimmedContent, 60,
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
	record.BodyContent = fields[0]
	record.DueDate, _ = time.Parse(EXPLICIT_TIME_FORMAT, fields[1])
	if fields[2] != "" {
		record.Repeat = &fields[2]
	}
	overdueDays, _ := strconv.ParseInt(fields[3], 10, 32)
	record.OverdueDays = int(overdueDays)
	record.Category = fields[4]
	record.DateCompleted, _ = time.Parse(RECORD_TIME_FORMAT, fields[5])

	if len(fields) >= 7 {
		record.Annotation = fields[6]
	}

	return record
}
