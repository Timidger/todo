package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path"
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
	due_date := record.Due_date.Format(EXPLICIT_TIME_FORMAT)
	repeat := ""
	if record.Repeat != nil {
		repeat = record.Repeat.String()
	}
	overdue_days := strconv.Itoa(record.Overdue_days)
	category := ""
	if record.Category != nil {
		category = *record.Category
	}
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
	result := fmt.Sprintf("%s\t%v", record.Body_content, record.DateCompleted.Format(EXPLICIT_TIME_FORMAT))

	date_due := record.Due_date.AddDate(0, 0, record.Overdue_days)
	if date_due.Before(record.DateCompleted) {
		overdue_days := int(math.Floor(record.DateCompleted.Sub(date_due).Hours() / 24))
		result += fmt.Sprintf(RED+"\toverdue %d days"+RESET, overdue_days)
	}
	return result
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
	if fields[4] != "" {
		record.Category = &fields[4]
	}
	record.DateCompleted, _ = time.Parse(EXPLICIT_TIME_FORMAT, fields[5])

	return record
}

func Fields() string {
	return "Body_content, Due_date, Repeat, Overdue_days, Category, DateCompleted" + "\n"
}

func FieldCount() int {
	return strings.Count(Fields(), ",") + 1
}

func AuditLog(manager *TaskManager, task Task) {
	audit_log_path := path.Join(manager.root_storage_directory, AUDIT_LOG)
	var audit_log_file *os.File
	if _, err := os.Stat(audit_log_path); os.IsNotExist(err) {
		audit_log_file, err = os.OpenFile(audit_log_path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}
		audit_log_file.WriteString("#" + Fields())
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

func AuditRecords(manager *TaskManager) Records {
	audit_log_path := path.Join(manager.root_storage_directory, AUDIT_LOG)
	if _, err := os.Stat(audit_log_path); os.IsNotExist(err) {
		LogError("There are no records yet")
		return Records{}
	}

	audit_log_file, err := os.OpenFile(audit_log_path, os.O_RDONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer audit_log_file.Close()

	audit_log := csv.NewReader(audit_log_file)
	audit_log.FieldsPerRecord = FieldCount()
	audit_log.Comment = '#'

	read_records, err := audit_log.ReadAll()
	if err != nil {
		panic(err)
	}
	var records Records
	for _, read_record := range read_records {
		records = append(records, Unmarshal(read_record))
	}
	return records
}
