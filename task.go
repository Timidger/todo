package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const CONTENT_LENGTH int = 75

type Task struct {
	// The "body" content of the task.
	body_content string
	// When this task is due and must be done.
	// If nil then there is no deadline
	due_date  *time.Time
	file_name string
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
	if len(task.body_content) < CONTENT_LENGTH {
		return fmt.Sprintf("%-10s%-80v%s", task.index+":", task.body_content, category_name)
	} else {
		words := strings.Split(task.body_content, " ")
		first := true
		result := ""
		buffer := ""
		for _, word := range words {
			// TODO Deal with empty buffer (e.g. words > CONTENT_LENGTH)
			if len(buffer)+len(word)+1 > CONTENT_LENGTH {
				if first {
					result = fmt.Sprintf("%-10s%-80v%s", task.index+":", buffer, category_name)
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
func NewTask(text string, due_date *time.Time) Task {
	if !utf8.ValidString(text) {
		panic(fmt.Sprintf("Invalid UTF-8 string: %v", text))
	}
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		LogError("Cannot make a task with an empty string")
		os.Exit(1)
	}
	var task Task
	task.body_content = text
	task.due_date = due_date
	return task
}

// Format just the task body
func (task *Task) FormatTask() string {
	if task.DueBeforeToday() {
		return RED + task.String() + RESET
	} else if task.DueAfter(time.Now().AddDate(0, 0, 6)) {
		return GREY + task.String() + RESET
	}
	return task.String()
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
