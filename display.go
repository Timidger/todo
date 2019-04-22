package main

import (
	"fmt"
	"time"
)

/// Displays tasks in the "Long" form, grouping tasks in distinct days and
/// greying out tasks beyond a week.
///
/// Non-deadline tasks are also displayed.
func DisplayTasksLong(tasks Tasks) {
	cur_day := tasks[0].due_date
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
		fmt.Printf("%-10s%s\n", task.index+":", task.FormatTask())
	}
	if no_deadlines {
		fmt.Printf(GREY + "No Deadline:" + RESET + "\n")
		for i, task := range tasks {
			if task.due_date == nil {
				fmt.Printf(GREY+"%d:\t%s"+RESET+"\n", i, task.FormatTask())
			}
		}
	}
}
