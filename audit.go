package main

import (
	"encoding/csv"
	"os"
	"path"
	"strconv"
	"time"
)

const AUDIT_LOG = "audit_log/"

// NOTE This is an append-only struct.
// If new fields are added, put them at the end.
type Record struct {
	Body_content  string
	Due_date      time.Time
	Repeat        *time.Duration
	Overdue_days  int
	Category      *string
	DateCompleted time.Time
}

func (record Record) Marshal() []string {
	body_content := record.Body_content
	due_date := record.Due_date.String()
	repeat := ""
	if record.Repeat != nil {
		repeat = record.Repeat.String()
	}
	overdue_days := strconv.Itoa(record.Overdue_days)
	category := ""
	if record.Category != nil {
		category = *record.Category
	}
	date_completed := record.DateCompleted.String()
	return []string{
		body_content,
		due_date,
		repeat,
		overdue_days,
		category,
		date_completed,
	}
}

func Fields() string {
	return "Body_content, Due_date, Repeat, Overdue_days, Category, DateCompleted" + "\n"
}

func AuditLog(task Task, manager *TaskManager) {
	audit_log_path := path.Join(manager.root_storage_directory, AUDIT_LOG)
	var audit_log_file *os.File
	if _, err := os.Stat(audit_log_path); os.IsNotExist(err) {
		audit_log_file, err = os.OpenFile(audit_log_path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}
		audit_log_file.WriteString(Fields())
	} else {
		audit_log_file, err = os.OpenFile(audit_log_path, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
	}

	defer audit_log_file.Close()

	audit_log := csv.NewWriter(audit_log_file)

	var record Record
	record.Body_content = task.Body_content
	record.Due_date = task.Due_date
	record.Repeat = task.Repeat
	record.Overdue_days = task.Overdue_days
	record.Category = task.category
	record.DateCompleted = time.Now()

	if err := audit_log.Write(record.Marshal()); err != nil {
		panic(err)
	}
	audit_log.Flush()
}
