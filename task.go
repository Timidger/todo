package todo

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

const CONTENT_LENGTH int = 58

const RECORD_TIME_FORMAT = "2006/01/02 MST 15:04:05"
const EXPLICIT_TIME_FORMAT = "2006/01/02 MST"
const RELATIVE_TIME_FORMAT = "Monday MST"

/*
 * XXX Yes these names are stupid but they are part of the JSON
 * and thus can't change now
 */

type Task struct {
	// The "body" content of the task.
	Body_content string
	// The first day when this task will appear. Not the actual due date.
	Due_date time.Time
	// When to repeat this task when it is deleted.
	// If it is null this task does not repeat.
	Repeat *time.Duration
	// How many days until this task is actually due.
	Overdue_days int
	file_name    string
	// The minimal index needed to specify this task
	index string
	// The full index
	full_index string
	// optional category
	category *string
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (task Task) GetFullIndex() string {
	return task.full_index
}

func (task Task) Category() string {
	if task.category != nil {
		return *task.category
	}
	return ""
}

func (task Task) String() string {
	category_name := ""
	if task.category != nil {
		category_name = "(" + *task.category + ")"
	}
	days_left := " "
	due_date := time.Now().AddDate(0, 0, -task.Overdue_days+1)
	if task.DueBefore(due_date) {
		passed_due_date := task.Due_date.AddDate(0, 0, task.Overdue_days).Truncate(24 * time.Hour)
		overdue_days := int(math.Floor(time.Now().Sub(passed_due_date).Hours() / 24))
		if task.DueBefore(time.Now().Truncate(24*time.Hour)) && overdue_days >= 0 {
			if overdue_days == 1 || overdue_days == 0 {
				days_left = fmt.Sprintf(" (%d day overdue)", 1)
			} else {
				days_left = fmt.Sprintf(" (%d days overdue)", overdue_days)
			}
		} else {
			days_left = " (due today)"
		}
	} else if task.Overdue_days > 0 {
		final_due_date := task.Due_date.AddDate(0, 0, task.Overdue_days)
		days_left = fmt.Sprintf(" (%d days left)",
			int(math.Ceil(final_due_date.Sub(time.Now()).Hours()/24)))
	}
	trimmed_content := strings.TrimSuffix(task.Body_content, "\n")
	preamble := task.index + ":"
	return HardWrapString(trimmed_content,
		60,
		preamble,
		10,
		fmt.Sprintf("%-15s%s", category_name, days_left),
		" ")
}

/// Creates a new task, without saving it.
func NewTask(text string, due_date time.Time, repeat *time.Duration, overdue_days int) (Task, error) {
	if !utf8.ValidString(text) {
		panic(fmt.Sprintf("Invalid UTF-8 string: %v", text))
	}
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		msg := "Cannot make a task with an empty string"
		LogError(msg)
		return Task{}, errors.New(msg)
	}
	var task Task
	task.Body_content = text
	task.Due_date = due_date
	task.Repeat = repeat
	task.Overdue_days = overdue_days
	return task, nil
}

// Format just the task body
func (task *Task) FormatTask() string {
	if task.DueBefore(time.Now().AddDate(0, 0, -task.Overdue_days)) {
		return RED + task.String() + RESET
	} else if task.DueAfter(time.Now().AddDate(0, 0, 6)) {
		return GREY + task.String() + RESET
	}
	return RESET + task.String()
}

/// Determines if a task is due exactly on this day. Not before, not after.
func (task *Task) DueOn(date time.Time) bool {
	if !task.Due_date.Before(date) || task.DueBefore(date) {
		return false
	}
	return true
}

// Determines if a task is due today (or any days before today)
//
// NOTE This is NOT a special case of Task.DueOn.
func (task *Task) DueToday() bool {
	return is_same_day(task.Due_date, time.Now()) || task.Due_date.Before(time.Now())
}

/// Determines if a task is due before today.
func (task *Task) DueBefore(date time.Time) bool {
	return task.Due_date.Before(date.Truncate(24 * time.Hour))
}

func (task *Task) DueAfter(after time.Time) bool {
	return task.Due_date.After(after)
}

func is_same_day(a time.Time, b time.Time) bool {
	return a.Day() == b.Day() && a.Month() == b.Month() && a.Year() == b.Year()
}
