package main

import (
	"fmt"
	"strconv"
	"time"
)

type Task struct {
	// The "body" content of the task.
	body_content string
	// When this task is due and must be done.
	// If nil then there is no deadline
	due_date  *time.Time
	file_name string
	// The minimal index needed to specify this task
	index string
}

func (task Task) String() string {
	return task.body_content
}

// Format just the task body
func (task *Task) FormatTask() string {
	format_string := "%v"
	if task.DueBeforeToday() {
		format_string = RED + format_string + RESET
	} else if task.DueAfter(time.Now().AddDate(0, 0, 6)) {
		format_string = GREY + format_string + RESET
	}
	return fmt.Sprintf(format_string, task.body_content)
}

// Determines if a task is due today (or any days before today)
func (task *Task) DueToday() bool {
	if task.due_date == nil {
		return false
	}
	return task.due_date.Before(time.Now())
}

/// Determines if a task is due before today.
func (task *Task) DueBeforeToday() bool {
	if task.due_date == nil {
		return false
	}
	now := time.Now()
	day := string(strconv.Itoa(now.Day()))
	month := string(strconv.Itoa(int(now.Month())))
	if now.Day() < 10 {
		day = "0" + day
	}
	if now.Month() < 10 {
		month = "0" + month
	}
	today, err := time.Parse(EXPLICIT_TIME_FORMAT,
		fmt.Sprintf("%v/%v/%v EDT", now.Year(), month, day))
	if err != nil {
		panic(err)
	}
	return task.due_date.Before(today)
}

func (task *Task) DueAfter(after time.Time) bool {
	if task.due_date == nil {
		return false
	}
	return task.due_date.After(after)
}
