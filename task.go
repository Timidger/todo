package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const CONTENT_LENGTH int = 58

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

func (task Task) String() string {
	category_name := ""
	if task.category != nil {
		category_name = "(" + *task.category + ")"
	}
	days_left := " "
	due_date := time.Now().AddDate(0, 0, -task.Overdue_days+1)
	if task.DueBefore(due_date) {
		passed_due_date := task.Due_date.AddDate(0, 0, task.Overdue_days)
		overdue_days := int(math.Floor(time.Now().Sub(passed_due_date).Hours() / 24))
		if overdue_days == 0 {
			days_left = " (due today)"
		} else {
			days_left = fmt.Sprintf(" (%d days overdue)", overdue_days)
		}
	} else if task.Overdue_days > 0 {
		final_due_date := task.Due_date.AddDate(0, 0, task.Overdue_days)
		days_left = fmt.Sprintf(" (%d days left)",
			int(math.Ceil(final_due_date.Sub(time.Now()).Hours()/24)))
	}
	trimmed_content := strings.TrimSuffix(task.Body_content, "\n")
	if len(task.Body_content) < CONTENT_LENGTH {
		return fmt.Sprintf("%-10s%-60v%-15s%s", task.index+":", trimmed_content, category_name, days_left)
	} else {
		words := strings.Split(trimmed_content, " ")
		first := true
		result := ""
		buffer := ""
		for _, word := range words {
			if len(buffer)+len(word)+1 > CONTENT_LENGTH {
				if first {
					result = fmt.Sprintf("%-10s%-60v%-15s%s",
						task.index+":", buffer, category_name, days_left)
					first = false
				} else {
					result += fmt.Sprintf("\n          %v", buffer)
				}
				buffer = ""
			}
			buffer += word + " "
		}
		if len(buffer) != 0 {
			result += fmt.Sprintf("\n          %v", buffer)
		}
		return result
	}
}

/// Creates a new task, without saving it.
func NewTask(text string, due_date time.Time, repeat *time.Duration, overdue_days int) Task {
	if !utf8.ValidString(text) {
		panic(fmt.Sprintf("Invalid UTF-8 string: %v", text))
	}
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		LogError("Cannot make a task with an empty string")
		os.Exit(1)
	}
	var task Task
	task.Body_content = text
	task.Due_date = due_date
	task.Repeat = repeat
	task.Overdue_days = overdue_days
	return task
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
	day := string(strconv.Itoa(date.Day()))
	month := string(strconv.Itoa(int(date.Month())))
	if date.Day() < 10 {
		day = "0" + day
	}
	if date.Month() < 10 {
		month = "0" + month
	}
	today, err := time.Parse(EXPLICIT_TIME_FORMAT,
		fmt.Sprintf("%v/%v/%v EDT", date.Year(), month, day))
	if err != nil {
		panic(err)
	}
	return task.Due_date.Before(today)
}

func (task *Task) DueAfter(after time.Time) bool {
	return task.Due_date.After(after)
}

func is_same_day(a time.Time, b time.Time) bool {
	return a.Day() == b.Day() && a.Month() == b.Month() && a.Year() == b.Year()
}
