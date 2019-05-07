package main

import (
	"fmt"
	"time"
)

const (
	GREY  = "\x1b[90m"
	RED   = "\x1b[31m"
	GREEN = "\x1b[32m"
	RESET = "\x1b[39m"
)

func LogSuccess(s string) {
	fmt.Printf(GREEN + s + RESET + "\n")
}

func LogError(s string) {
	fmt.Printf(RED + s + RESET + "\n")
}

/// Displays tasks in the "Short" form. Just a list of unique hashes and
/// the body contents. Really only useful for showing "today's" tasks.
func DisplayTasks(tasks Tasks) {
	for _, task := range tasks {
		fmt.Println(task.FormatTask())
	}
}

/// Displays tasks in the "Long" form, grouping tasks in distinct days and
/// greying out tasks beyond a week.
///
/// Non-deadline tasks are also displayed.
func DisplayTasksLong(tasks Tasks) {
	var cur_day *time.Time = nil
	for _, task := range tasks {
		if task.due_date != nil {
			cur_day = task.due_date
			break
		}
	}
	no_deadlines := false
	for i, task := range tasks {
		if cur_day == nil || task.due_date == nil {
			no_deadlines = true
			continue
		}
		if i == 0 || !cur_day.Equal(*task.due_date) {
			cur_day = task.due_date
			day_header := fmt.Sprintf("%-40v\t%v\n",
				cur_day.Format("Monday")+":",
				cur_day.Format(EXPLICIT_TIME_FORMAT))
			if task.DueBeforeToday() {
				day_header = RED + day_header + RESET
			} else if task.DueAfter(time.Now().AddDate(0, 0, 6)) {
				day_header = GREY + day_header + RESET
			}
			fmt.Printf(day_header)
		}
		fmt.Println(task.FormatTask())
	}
	if no_deadlines {
		fmt.Printf(GREY + "No Deadline:" + RESET + "\n")
		for _, task := range tasks {
			if task.due_date == nil {
				fmt.Printf(GREY+"%s"+RESET+"\n", task.FormatTask())
			}
		}
	}
}
