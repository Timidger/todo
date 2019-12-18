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
const DAY_HOURS = time.Hour * 24

const RECORD_TIME_FORMAT = "2006/01/02 MST 15:04:05"
const EXPLICIT_TIME_FORMAT = "2006/01/02 MST"
const RELATIVE_TIME_FORMAT = "Monday MST"

type Task struct {
	// The "body" content of the task.
	BodyContent string
	// The first day when this task will appear. Not the actual due date.
	DueDate time.Time
	// When to repeat this task when it is deleted.
	// If it is null this task does not repeat.
	//
	// Can either be a time.Duration (as an int) or a more flexible date
	// representation (e.g. Monday or a list of days with , separating).
	Repeat *string
	// How many days until this task is actually due.
	OverdueDays int
	fileName    string
	// The minimal index needed to specify this task
	index string
	// The full index
	fullIndex string
	// optional category
	category *string
}

func (task Task) GetFullIndex() string {
	return task.fullIndex
}

func (task Task) Category() string {
	if task.category != nil {
		return *task.category
	}
	return ""
}

func (task Task) String() string {
	categoryName := ""
	if task.category != nil {
		categoryName = "(" + *task.category + ")"
	}
	daysLeft := " "
	dueDate := time.Now().AddDate(0, 0, -task.OverdueDays)
	if task.DueBefore(dueDate) {
		passedDueDate := task.DueDate.AddDate(0, 0, task.OverdueDays)
		passedDueDate = passedDueDate.Truncate(DAY_HOURS)
		overdueDays := int(math.Floor(time.Now().Sub(passedDueDate).Hours() / 24))
		if task.DueBefore(time.Now().Truncate(DAY_HOURS)) && overdueDays >= 0 {
			if overdueDays == 1 || overdueDays == 0 {
				daysLeft = fmt.Sprintf(" (%d day overdue)", 1)
			} else {
				daysLeft = fmt.Sprintf(" (%d days overdue)", overdueDays)
			}
		} else {
			daysLeft = " (due today)"
		}
	} else if task.OverdueDays > 0 {
		finalDueDate := task.DueDate.AddDate(0, 0, task.OverdueDays)
		days := int(math.Ceil(finalDueDate.Sub(time.Now()).Hours() / 24))
		if days == 0 {
			daysLeft = " (due today)"
		} else if days == 1 {
			daysLeft = fmt.Sprintf(" (%d day left)", days)
		} else {
			daysLeft = fmt.Sprintf(" (%d days left)", days)
		}
	}
	trimmed_content := strings.TrimSuffix(task.BodyContent, "\n")
	preamble := task.index + ":"
	return HardWrapString(trimmed_content,
		60,
		preamble,
		10,
		fmt.Sprintf("%-15s%s", categoryName, daysLeft),
		" ")
}

/// Creates a new task, without saving it.
func NewTask(text string, dueDate time.Time, repeat *string, overdueDays int) (Task, error) {
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
	task.BodyContent = text
	task.DueDate = dueDate
	task.Repeat = repeat
	task.OverdueDays = overdueDays
	return task, nil
}

// Format just the task body
func (task *Task) FormatTask() string {
	if task.DueBefore(time.Now().AddDate(0, 0, -task.OverdueDays)) {
		return RED + task.String() + RESET
	} else if task.DueAfter(time.Now().AddDate(0, 0, 6)) {
		return GREY + task.String() + RESET
	}
	return RESET + task.String()
}

/// Determines if a task is due exactly on this day. Not before, not after.
func (task *Task) DueOn(date time.Time) bool {
	return !(task.DueDate.Before(date) || task.DueBefore(date))
}

// Determines if a task is due today (or any days before today)
//
// NOTE This is NOT a special case of Task.DueOn.
func (task *Task) DueToday() bool {
	return is_same_day(task.DueDate, time.Now()) ||
		task.DueDate.Before(time.Now())
}

/// Determines if a task is due before today.
func (task *Task) DueBefore(date time.Time) bool {
	return task.DueDate.Before(date.Truncate(DAY_HOURS))
}

func (task *Task) DueAfter(after time.Time) bool {
	return task.DueDate.After(after)
}

func is_same_day(a time.Time, b time.Time) bool {
	return a.Day() == b.Day() &&
		a.Month() == b.Month() &&
		a.Year() == b.Year()
}
