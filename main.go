package main

import (
	"bufio"
	"fmt"
	"git.sr.ht/~sircmpwn/getopt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

const help_message = "Usage of todo:\n" +
	"  -h              Show this help message\n" +
	"  -l              List the things to do today in no particular order\n" +
	"  -a              List all the things to do, regardless of due date, from soonest to latest\n" +
	"  -d <index>      Delete a task by index number. If preceded by -a based on full list, not just today\n" +
	"  -x <index>      Delay a task by one day. It is suggested you don't do this too often\n" +
	"  -t <date>       Delay the task until the date\n" +
	"                  Date uses YYYY/MM/DD. Relative days such as \"Monday\" or \"Tomorrow\" are also supported\n" +
	"  -c <category>   Specify a category\n" +
	"  -C <category>   Create a new category\n" +
	"  -L              List all the categories\n" +
	"  -n              Don't have a deadline for this task\n" +
	"                  Note that todos with no deadline only appear with -a\n" +
	"  -D <directory>  Specify a custom todo directory (default is ~/.todo)\n"

const EXPLICIT_TIME_FORMAT = "2006/01/02 MST"
const RELATIVE_TIME_FORMAT = "Monday MST"

const (
	LISTING_ALL = iota
	LISTING_TODAY
)

func main() {
	opts, others, err := getopt.Getopts(os.Args[1:], "nLhalt:d:x:D:C:c:")
	if err != nil {
		panic(err)
	}
	var manager TaskManager
	manager.storage_directory = path.Join(os.Getenv("HOME"), ".todo/")
	now := time.Now()
	due_date := &now
	skip_task_read := false
	listing := LISTING_TODAY
	for _, opt := range opts {
		switch opt.Option {
		case 't':
			if due_date == nil {
				due_date = &now
			}
			*due_date, err = time.Parse(EXPLICIT_TIME_FORMAT, opt.Value+" EDT")
			if err != nil {
				*due_date = time.Now()
				relative_day := 0
				cur_weekday := int(due_date.Weekday())
				switch strings.Title(opt.Value) {
				case "Sunday":
					relative_day = 0
				case "Monday":
					relative_day = 1
				case "Tuesday":
					relative_day = 2
				case "Wednesday":
					relative_day = 3
				case "Thursday":
					relative_day = 4
				case "Friday":
					relative_day = 5
				case "Saturday":
					relative_day = 6
				case "Tomorrow":
					relative_day = (cur_weekday + 1) % 7
				default:
					LogError(fmt.Sprintf("Bad date: %s", opt.Value))
					os.Exit(1)
				}
				if cur_weekday < relative_day {
					*due_date = due_date.AddDate(0, 0, int(relative_day-cur_weekday))
				} else {
					*due_date = due_date.AddDate(0, 0, 7-(cur_weekday-relative_day))
				}
			}
		case 'h':
			fmt.Printf(help_message)
			return
		case 'l':
			listing = LISTING_TODAY
			skip_task_read = true
			tasks := manager.GetTasksToday()
			if len(tasks) == 0 {
				tasks = manager.GetTasks()
				if len(tasks) != 0 {
					DisplayTasksLong(tasks)
				}
				break
			}
			DisplayTasks(tasks)
		case 'a':
			listing = LISTING_ALL
			skip_task_read = true
			tasks := manager.GetTasks()
			if len(tasks) == 0 {
				break
			}
			DisplayTasksLong(tasks)
		case 'd':
			skip_task_read = true
			index := opt.Value
			var task_deleted *Task
			switch listing {
			case LISTING_ALL:
				tasks := manager.GetTasks()
				task_deleted = manager.DeleteTask(tasks, index)
			case LISTING_TODAY:
				tasks := manager.GetTasksToday()
				task_deleted = manager.DeleteTask(tasks, index)
				// Hack to get around the coloration display
				if task_deleted != nil {
					task_deleted.due_date = &now
				}
			default:
				panic(fmt.Sprintf("Unknown flag %v", listing))
			}
			if task_deleted == nil {
				LogError(fmt.Sprintf("Bad index \"%s\"", index))
				os.Exit(1)
			}
			LogSuccess(task_deleted.String())
		case 'x':
			skip_task_read = true
			index := opt.Value
			var tasks []Task
			switch listing {
			case LISTING_ALL:
				tasks = manager.GetTasks()
			case LISTING_TODAY:
				tasks = manager.GetTasksToday()
			}
			task_removed := manager.DeleteTask(tasks, index)
			if task_removed == nil {
				LogError(fmt.Sprintf("Bad index\"%s\"", index))
				os.Exit(1)
			}
			if task_removed.due_date == nil {
				manager.AddTask(task_removed.body_content, nil)
				LogError("Cannot delay a todo with no deadline!")
				os.Exit(1)
			}
			new_date := task_removed.due_date.AddDate(0, 0, 1)
			manager.AddTask(task_removed.body_content, &new_date)
			LogError(fmt.Sprintf("Task \"%s\" delayed until %s",
				task_removed.body_content, new_date.Weekday()))
		case 'D':
			manager.storage_directory = opt.Value
		case 'c':
			category := opt.Value
			category_path := path.Join(manager.storage_directory, category)
			if _, err := os.Stat(category_path); os.IsNotExist(err) {
				LogError(fmt.Sprintf("Category \"%s\" does not exist", category))
				os.Exit(1)
			}
			fallthrough
		case 'C':
			manager.storage_directory = path.Join(manager.storage_directory, opt.Value)
		case 'L':
			skip_task_read = true
			categories := manager.GetCategories()
			for _, category := range categories {
				fmt.Println(category)
			}
		case 'n':
			due_date = nil
		}

	}
	if len(opts) > 0 && skip_task_read {
		return
	}
	if input := strings.Join(os.Args[others+1:], " "); len(os.Args) > 1 && input != "" {
		manager.AddTask(input, due_date)
	} else {
		reader := bufio.NewReader(os.Stdin)
		manager.AddTask(readInTask(reader), due_date)
	}
}

// Read a task in from a reader and pass it off to add_task.
func readInTask(reader *bufio.Reader) string {
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	text := string(bytes)
	return text
}
