package main

import (
	"encoding/json"
	"os"
	"path"
	"time"
)

type Record struct {
	Body_content  string
	Due_date      time.Time
	Repeat        *time.Duration
	Overdue_days  int
	Category      *string
	DateCompleted time.Time
}

const AUDIT_LOG = "audit_log/"

func AuditLog(task Task, manager *TaskManager) {
	audit_log_path := path.Join(manager.root_storage_directory, AUDIT_LOG)

	audit_log, err := os.OpenFile(audit_log_path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	defer audit_log.Close()

	var record Record
	record.Body_content = task.Body_content
	record.Due_date = task.Due_date
	record.Repeat = task.Repeat
	record.Overdue_days = task.Overdue_days
	record.Category = task.category
	record.DateCompleted = time.Now()

	record_json, err := json.Marshal(record)
	if err != nil {
		panic(err)
	}

	if _, err = audit_log.Write(record_json); err != nil {
		panic(err)
	}

	if _, err = audit_log.WriteString("\n"); err != nil {
		panic(err)
	}
}
