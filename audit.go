package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const AUDIT_LOG = "audit_log/"

type Records []Record

func (records Records) Len() int {
	return len(records)
}

func (records Records) Less(i, j int) bool {
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
	date_completed := record.DateCompleted.Format(EXPLICIT_TIME_FORMAT)
	return []string{
		body_content,
		due_date,
		repeat,
		overdue_days,
		category,
		date_completed,
	}
}

func (record Record) String() string {
	completed := record.DateCompleted.Format(EXPLICIT_TIME_FORMAT)
	overdue := ""
	date_due := record.Due_date.AddDate(0, 0, record.Overdue_days)
	if date_due.Before(record.DateCompleted) {
		overdue_days := int(math.Floor(record.DateCompleted.Sub(date_due).Hours() / 24))
		overdue = fmt.Sprintf(RED+" (overdue %d days)"+RESET, overdue_days)
	}

	category_name := ""
	if record.Category != "" {
		category_name = "(" + record.Category + ")"
	}

	trimmed_content := strings.TrimSuffix(record.Body_content, "\n")
	if len(record.Body_content) < CONTENT_LENGTH {
		return fmt.Sprintf("%s  %-60v%-15s%s", completed, trimmed_content, category_name, overdue)
	} else {
		words := strings.Split(trimmed_content, " ")
		first := true
		result := ""
		buffer := ""
		for _, word := range words {
			if len(buffer)+len(word)+1 > CONTENT_LENGTH {
				if first {
					result = fmt.Sprintf("%s  %-60v%-15s%s",
						completed, buffer, category_name, overdue)
					first = false
				} else {
					result += fmt.Sprintf("\n                %v", buffer)
				}
				buffer = ""
			}
			buffer += word + " "
		}
		if len(buffer) != 0 {
			result += fmt.Sprintf("\n                %v", buffer)
		}
		return result
	}
}

func Unmarshal(fields []string) Record {
	if len(fields) != FieldCount() {
		panic(fmt.Sprintf("actual %d != expected %d", len(fields), FieldCount()))
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
	record.DateCompleted, _ = time.Parse(EXPLICIT_TIME_FORMAT, fields[5])

	return record
}

func Fields() string {
	return "Body_content, Due_date, Repeat, Overdue_days, Category, DateCompleted" + "\n"
}

func FieldCount() int {
	return strings.Count(Fields(), ",") + 1
}
