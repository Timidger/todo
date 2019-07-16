package main

import (
	"bufio"
	"fmt"
	"git.sr.ht/~sircmpwn/getopt"
	"github.com/mattn/go-isatty"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const help_message = "Usage of todo:\n" +
	"  -h              Show this help message\n" +
	"  -l              List the things to do today by category first and due date\n" +
	"  -a              List all the things to do, regardless of due date, from soonest to latest\n" +
	"  -d <index>      Delete a task by index number. If preceded by -a based on full list, not just due now\n" +
	"  -D <index>      Same as -d but it will not recreate a task that repeats\n" +
	"  -x <index>      Delay a task by one day. It is suggested you don't do this too often\n" +
	"  -t <date>       Delay the task until the date\n" +
	"                  Date uses YYYY/MM/DD. Relative days such as \"Monday\" or \"Tomorrow\" are also supported\n" +
	"  -r <number>     Repeat this task after a number of days. Based on the due date, not the day it was deleted\n" +
	"                  Default 0, Must be positive.\n" +
	"  -n <number>     Days until this task is actually due. Think of this as \"How many days I want to work on this task\"\n" +
	"                  Default 0, must be positive\n" +
	"  -c <category>   Specify a category\n" +
	"  -C <category>   Create a new category\n" +
	"  -L              List all the categories\n" +
	"  -S <directory>  Specify a custom todo directory (default is ~/.todo). Primarily used for testing\n"

const EXPLICIT_TIME_FORMAT = "2006/01/02 MST"
const RELATIVE_TIME_FORMAT = "Monday MST"

const (
	LISTING_ALL = iota
	LISTING_DAY
)

var tasks *Tasks = nil

func get_tasks(manager *TaskManager) *Tasks {
	if tasks == nil {
		tasks_ := manager.GetTasks()
		tasks = &tasks_
	}
	return tasks
}

func main() {
	opts, others, err := getopt.Getopts(os.Args, "zLhalt:d:x:D:S:C:c:r:n:")
	if err != nil {
		panic(err)
	}
	var manager TaskManager
	manager.storage_directory = path.Join(os.Getenv("HOME"), ".todo/")
	manager.root_storage_directory = manager.storage_directory
	overdue_days := 0
	due_date := time.Now()
	var repeat *time.Duration = nil
	skip_task_read := false
	force_delete := false
	listing := LISTING_DAY
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
					LogError(fmt.Sprintf("Bad date: %s", opt.Value))
					os.Exit(1)
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
			all_tasks := get_tasks(&manager)
			listing = LISTING_DAY
			skip_task_read = true
			is_today := true
			var tasks Tasks
			if due_date.Before(time.Now()) {
				tasks = all_tasks.FilterTasksDueBeforeToday()
				if len(tasks) == 0 {
					tasks = *all_tasks
					if len(tasks) != 0 {
						DisplayTasksLong(tasks)
					}
					break
				}
			} else {
				is_today = false
				tasks = all_tasks.FilterTasksDueOnDay(due_date)
			}
			if is_today {
				DisplayTasks(tasks)
			} else {
				DisplayTasksLong(tasks)
			}
		case 'a':
			// We do the listing down below, this is so if this is
			// paired with actions like delete we don't print everything to
			// stdout for sure. This is to allow piping with that to work with
			// e.g. todo -ac new_category $(todo -d 1234)
			listing = LISTING_ALL
		case 'D':
			force_delete = true
			fallthrough
		case 'd':
			all_tasks := get_tasks(&manager)
			skip_task_read = true
			index := opt.Value
			var task_deleted *Task
			switch listing {
			case LISTING_DAY:
				var tasks Tasks
				if due_date.Before(time.Now()) {
					// NOTE This is a special case: we want everything due today
					// or before today with this call..
					tasks = all_tasks.FilterTasksDueBeforeToday()
				} else {
					tasks = all_tasks.FilterTasksDueOnDay(due_date)
				}
				if len(tasks) != 0 {
					task_deleted = manager.DeleteTask(tasks, index)
					if task_deleted != nil && task_deleted.Repeat == nil {
						all_tasks.RemoveFirst(*task_deleted)
					}
					break
				}
				// If there are no tasks today then we must try to delete based
				// on all tasks. This lets you use it like -l when there are no
				// tasks today.
				fallthrough
			case LISTING_ALL:
				task_deleted = manager.DeleteTask(*all_tasks, index)
				if task_deleted != nil {
					all_tasks.RemoveFirst(*task_deleted)
				}
			default:
				panic(fmt.Sprintf("Unknown flag %v", listing))
			}
			// If we delete the same task multiple times we'll need to reload,
			// so try doing that before giving up.
			if task_deleted == nil {
				LogError(fmt.Sprintf("Bad index \"%s\"", index))
				os.Exit(1)
			}
			if !isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Printf(task_deleted.Body_content)
			} else {
				LogSuccess(task_deleted.String())
			}
			if !force_delete && task_deleted.Repeat != nil {
				// Recreate the task if it has a repeat.
				task_deleted.Due_date = task_deleted.Due_date.Add(*task_deleted.Repeat)
				if err := manager.SaveTask(*task_deleted); err != nil {
					LogError(err.Error())
					os.Exit(1)
				}
			}
			// Log in the audit log
			if !force_delete {
				AuditLog(&manager, *task_deleted)
			}
			force_delete = false
		case 'r':
			days, err := strconv.ParseInt(opt.Value, 10, 32)
			if err != nil {
				LogError(fmt.Sprintf("Bad delay time: %s", opt.Value))
				os.Exit(1)
			}
			if days <= 0 {
				LogError("Delay time must be a positive, non-zero number")
				os.Exit(1)
			}
			hours, err := time.ParseDuration(fmt.Sprintf("%dh", days*24))
			if err != nil {
				panic(err)
			}
			repeat = &hours
		case 'x':
			all_tasks := get_tasks(&manager)
			skip_task_read = true
			index := opt.Value
			var tasks Tasks
			switch listing {
			case LISTING_ALL:
				tasks = *all_tasks
			case LISTING_DAY:
				tasks = all_tasks.FilterTasksDueBeforeToday()
			}
			task_deleted := manager.DeleteTask(tasks, index)
			if task_deleted == nil {
				LogError(fmt.Sprintf("Bad index \"%s\"", index))
				os.Exit(1)
			}
			task_deleted.Due_date = task_deleted.Due_date.AddDate(0, 0, 1)
			*all_tasks.GetByHash(index) = *task_deleted
			if err := manager.SaveTask(*task_deleted); err != nil {
				LogError(err.Error())
				os.Exit(1)
			}
			(fmt.Printf("Task \"%s\" delayed until %s\n",
				task_deleted.Body_content, task_deleted.Due_date.Weekday()))
		case 'S':
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
			days, err := strconv.ParseInt(opt.Value, 10, 32)
			if err != nil {
				LogError(fmt.Sprintf("Bad day delay \"%s\", need number", opt.Value))
				os.Exit(1)
			}
			if days <= 0 {
				LogError("Delay time must be a positive number")
				os.Exit(1)
			}
			overdue_days = int(days)
		case 'z':
			skip_task_read = true
			records := AuditRecords(&manager)
			for _, record := range records {
				fmt.Printf("%s\n", record.String())
			}
		}

	}
	// if skip_task_read is set then we have done an action that means we should
	// not print this to stdout due to the chaining rule described above.
	if listing == LISTING_ALL && !skip_task_read {
		all_tasks := get_tasks(&manager)
		if len(*all_tasks) != 0 {
			DisplayTasksLong(*all_tasks)
		}
		skip_task_read = true
	}
	if len(opts) > 0 && skip_task_read {
		return
	}
	if input := strings.Join(os.Args[others:], " "); len(os.Args) > 1 && input != "" {
		err = manager.SaveTask(NewTask(input, due_date, repeat, overdue_days))
	} else {
		reader := bufio.NewReader(os.Stdin)
		err = manager.SaveTask(NewTask(readInTask(reader), due_date, repeat, overdue_days))
	}
	if err != nil {
		LogError(err.Error())
		os.Exit(1)
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
