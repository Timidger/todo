package main

import (
	"bufio"
	"fmt"
	"git.sr.ht/~sircmpwn/getopt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
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
	"  -D <directory>  Specify a custom todo directory (default is ~/.todo)\n"

const EXPLICIT_TIME_FORMAT = "2006/01/02 MST"
const RELATIVE_TIME_FORMAT = "Monday MST"
const (
	GREY  = "\x1b[90m"
	RED   = "\x1b[31m"
	GREEN = "\x1b[32m"
	RESET = "\x1b[39m"
)

const (
	LISTING_ALL = iota
	LISTING_TODAY
)

func main() {
	opts, others, err := getopt.Getopts(os.Args[1:], "halt:d:x:D:")
	if err != nil {
		panic(err)
	}
	var manager TaskManager
	manager.storage_directory = path.Join(os.Getenv("HOME"), ".todo/")
	due_date := time.Now()
	skip_task_read := false
	listing := LISTING_TODAY
	for _, opt := range opts {
		switch opt.Option {
		case 't':
			due_date, err = time.Parse(EXPLICIT_TIME_FORMAT, opt.Value+" EDT")
			if err != nil {
				due_date = time.Now()
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
					panic(err)
				}
				if cur_weekday < relative_day {
					due_date = due_date.AddDate(0, 0, int(relative_day-cur_weekday))
				} else {
					due_date = due_date.AddDate(0, 0, 7-(cur_weekday-relative_day))
				}
			}
		case 'h':
			fmt.Printf(help_message)
			return
		case 'l':
			listing = LISTING_TODAY
			skip_task_read = true
			tasks := manager.GetTasks()
			i := 0
			for _, task := range tasks {
				if task.DueToday() {
					fmt.Printf("%d: %s\n", i, task.FormatTask())
					i++
				}
			}
		case 'a':
			listing = LISTING_ALL
			skip_task_read = true
			tasks := manager.GetTasks()
			if len(tasks) == 0 {
				break
			}
			cur_day := tasks[0].due_date
			for i, task := range tasks {
				if i == 0 || cur_day != task.due_date {
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
				fmt.Printf("%d:\t%s\n", i, task.FormatTask())
			}
		case 'd':
			skip_task_read = true
			to_delete, err := strconv.ParseInt(opt.Value, 10, 64)
			if err != nil {
				panic(err)
			}
			index := int(to_delete)
			switch listing {
			case LISTING_ALL:
				tasks := manager.GetTasks()
				task_deleted := manager.DeleteTask(tasks, index)
				fmt.Printf(GREEN+"%d: %s"+RESET+"\n", index, task_deleted.FormatTask())
			case LISTING_TODAY:
				tasks := manager.GetTasksToday()
				task_deleted := manager.DeleteTask(tasks, index)
				// Hack to get around the coloration display
				task_deleted.due_date = time.Now()
				fmt.Printf(GREEN+"%d: %s"+RESET+"\n", index, task_deleted.FormatTask())
			default:
				panic(fmt.Sprintf("Unknown flag %v", listing))
			}
		case 'x':
			skip_task_read = true
			to_delay, err := strconv.ParseInt(opt.Value, 10, 64)
			if err != nil {
				panic(err)
			}
			index := int(to_delay)
			var tasks []Task
			switch listing {
			case LISTING_ALL:
				tasks = manager.GetTasks()
			case LISTING_TODAY:
				tasks = manager.GetTasksToday()
			}
			task_removed := manager.DeleteTask(tasks, index)
			new_date := task_removed.due_date.AddDate(0, 0, 1)
			manager.AddTask(task_removed.body_content, new_date)
			fmt.Printf(RED+"Task \"%s\" delayed until %s"+RESET+"\n",
				task_removed.FormatTask(), new_date.Weekday())
		case 'D':
			manager.storage_directory = opt.Value
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
