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
	curDay := tasks[0].DueDate
	printed := false
	for _, task := range tasks {
		if !printed || !is_same_day(curDay, task.DueDate) {
			printed = true
			curDay = task.DueDate
			dayHeader := fmt.Sprintf("%-90v%v\n",
				curDay.Format("Monday")+":",
				curDay.Format(EXPLICIT_TIME_FORMAT))
			if task.DueAfter(time.Now().AddDate(0, 0, 6)) {
				dayHeader = GREY + dayHeader + RESET
			}
			fmt.Printf(dayHeader)
		}
		fmt.Println(task.FormatTask())
	}
}

/// Hard wraps a string to max_length. postamble will always be on the
/// first line after the max_length content (postamble is not hard wrapped).
///
/// Each paragraph will be indented at least preamble_length amount.
/// Pre-amble will be on the first line only.
func HardWrapString(paragraph string, maxLength int,
	preamblePart string, preambleLength int,
	postamble string, everyLinePreamble string) string {
	append_header := func(body string) string {
		return fmt.Sprintf("%-*s%-*v%s",
			preambleLength,
			preamblePart,
			maxLength,
			body,
			postamble)
	}

	preamble := fmt.Sprintf("%*s", preambleLength, everyLinePreamble)

	if len(paragraph) < maxLength {
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
