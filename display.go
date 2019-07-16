package main

import (
	"fmt"
	"os"
	"time"
)

const (
	GREY  = "\x1b[90m"
	RED   = "\x1b[31m"
	GREEN = "\x1b[32m"
	RESET = "\x1b[37m"
)

func LogSuccess(s string) {
	fmt.Fprintf(os.Stderr, GREEN+s+RESET+"\n")
}

func LogError(s string) {
	fmt.Fprintf(os.Stderr, RED+s+RESET+"\n")
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
	if len(tasks) == 0 {
		return
	}
	// NOTE This is the first day because it's assumed to be sorted.
	cur_day := tasks[0].Due_date
	printed := false
	for _, task := range tasks {
		if !printed || !is_same_day(cur_day, task.Due_date) {
			printed = true
			cur_day = task.Due_date
			day_header := fmt.Sprintf("%-90v%v\n",
				cur_day.Format("Monday")+":",
				cur_day.Format(EXPLICIT_TIME_FORMAT))
			if task.DueAfter(time.Now().AddDate(0, 0, 6)) {
				day_header = GREY + day_header + RESET
			}
			fmt.Printf(day_header)
		}
		fmt.Println(task.FormatTask())
	}
}
