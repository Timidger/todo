package todo

import (
	"fmt"
	"os"
	"strings"
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

/// Hard wraps a string to max_length. postamble will always be on the
/// first line after the max_length content (postamble is not hard wrapped).
///
/// Each paragraph will be indented at least preamble_length amount.
/// Pre-amble will be on the first line only.
func HardWrapString(paragraph string, max_length int,
	preamble_part string, preamble_length int,
	postamble string) string {
	append_header := func(body string) string {
		return fmt.Sprintf("%-*s%-*v%s",
			preamble_length,
			preamble_part,
			max_length,
			body,
			postamble)
	}

	preamble := fmt.Sprintf("%-*s", preamble_length, " ")

	if len(paragraph) < max_length {
		return append_header(paragraph)
	}

	words := strings.Split(paragraph, " ")
	first := true
	result := ""
	buffer := ""
	for _, word := range words {
		if len(buffer)+len(word)+1 > CONTENT_LENGTH {
			if first {
				result = append_header(buffer)
				first = false
			} else {
				result += "\n" + preamble + buffer
			}
			buffer = ""
		}
		buffer += word + " "
	}

	if len(buffer) != 0 {
		result += "\n" + preamble + buffer
	}
	return result
}
