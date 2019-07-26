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
	"                  You can also combine it with -t, I guess. Quitter\n" +
	"  -s <index>      Skip a task, deleting it but not logging it. This is only valid for repeat tasks.\n" +
	"                  Note that the task will be regenerated, if that's not what you want see -D\n" +
	"  -t <date>       Delay the task until the date\n" +
	"                  Date uses YYYY/MM/DD. Relative days such as \"Monday\" or \"Tomorrow\" are also supported\n" +
	"                  If coupled with -A then it will show logs of any events on or after this date\n" +
	"  -r <number>     Repeat this task after a number of days. Based on the due date, not the day it was deleted\n" +
	"                  Default 0, Must be positive.\n" +
	"  -n <number>     Days until this task is actually due. Think of this as \"How many days I want to work on this task\"\n" +
	"                  Default 0, must be positive\n" +
	"  -c <category>   Specify a category\n" +
	"  -C <category>   Create a new category\n" +
	"  -L              List all the categories\n" +
	"  -A              Show audit logs. Can be controlled with -t and -c\n" +
	"  -S <directory>  Specify a custom todo directory (default is ~/.todo). Primarily used for testing\n"

const RECORD_TIME_FORMAT = "2006/01/02 MST 15:04:05"
const EXPLICIT_TIME_FORMAT = "2006/01/02 MST"
const RELATIVE_TIME_FORMAT = "Monday MST"

func main() {
	opts, others, err := getopt.Getopts(os.Args, "ALhalt:d:x:D:S:C:c:r:n:s:")
	if err != nil {
		fmt.Printf("%s", help_message)
		return
	}
	var task_manager TaskManager
	task_manager.storage_directory = path.Join(os.Getenv("HOME"), ".todo/")

	var cmd_manager CommandManager
	cmd_manager.due_date = time.Now()
	cmd_manager.listing = LISTING_DAY

	for _, opt := range opts {
		switch opt.Option {
		case 'h':
			fmt.Printf("%s", help_message)
			return
		case 't':
			due_date, err := time.Parse(EXPLICIT_TIME_FORMAT, opt.Value+" EDT")
			if err == nil {
				cmd_manager.set_due_date(due_date)
			} else {
				err = cmd_manager.set_due_date_relative(opt.Value)
				if err != nil {
					LogError(err.Error())
					os.Exit(1)
				}
			}
		case 'l':
			tasks, err := cmd_manager.get_tasks(&task_manager)
			if err != nil {
				LogError(err.Error())
				os.Exit(1)
			}

			// TODO DisplayTasksLong?
			DisplayTasks(tasks)
		case 'a':
			cmd_manager.use_all_tasks()
		case 'D':
			task_deleted, err := cmd_manager.delete_task(&task_manager, opt.Value, true)
			if err != nil {
				LogError(err.Error())
				os.Exit(1)
			}
			if !isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Printf(task_deleted.Body_content)
			} else {
				LogSuccess(task_deleted.String())
			}
		case 'd':
			task_deleted, err := cmd_manager.delete_task(&task_manager, opt.Value, false)
			if err != nil {
				LogError(err.Error())
				os.Exit(1)
			}
			if !isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Printf(task_deleted.Body_content)
			} else {
				LogSuccess(task_deleted.String())
			}
		case 's':
			cmd_manager.skip_task(&task_manager, opt.Value)
		case 'r':
			days, err := strconv.ParseInt(opt.Value, 10, 32)
			if err != nil {
				LogError(fmt.Sprintf("Bad delay time: %s", opt.Value))
				os.Exit(1)
			}

			err = cmd_manager.set_repeat(int(days))
			if err != nil {
				LogError(err.Error())
				os.Exit(1)
			}
		case 'x':
			err := cmd_manager.delay_task(&task_manager, opt.Value)
			if err != nil {
				LogError(err.Error())
				os.Exit(1)
			}
		case 'S':
			task_manager.storage_directory = opt.Value
		case 'c':
			category := opt.Value
			category_path := path.Join(task_manager.storage_directory, category)
			if _, err := os.Stat(category_path); os.IsNotExist(err) {
				LogError(fmt.Sprintf("Category \"%s\" does not exist", category))
				os.Exit(1)
			}
			fallthrough
		case 'C':
			task_manager.storage_directory = path.Join(task_manager.storage_directory, opt.Value)
		case 'L':
			categories := cmd_manager.get_categories(&task_manager)
			for _, category := range categories {
				fmt.Println(category)
			}
		case 'n':
			days, err := strconv.ParseInt(opt.Value, 10, 32)
			if err != nil {
				LogError(fmt.Sprintf("Bad day delay \"%s\", need number", opt.Value))
				os.Exit(1)
			}

			err = cmd_manager.set_delay(int(days))
			if err != nil {
				LogError(err.Error())
				os.Exit(1)
			}
		case 'A':
			records := cmd_manager.get_audit_log(&task_manager)
			for _, record := range records {
				fmt.Println(record.String())
			}
		}

	}

	// if skip_task_read is set then we have done an action that means we should
	// not print this to stdout due to the chaining rule described above.
	tasks_all := cmd_manager.get_tasks_if_all(&task_manager)
	DisplayTasksLong(tasks_all)

	if len(opts) > 0 && cmd_manager.skip_task_creation_prompt {
		return
	}

	if input := strings.Join(os.Args[others:], " "); len(os.Args) > 1 && input != "" {
		err = cmd_manager.create_task(&task_manager, input)
	} else {
		reader := bufio.NewReader(os.Stdin)
		err = cmd_manager.create_task(&task_manager, readInTask(reader))
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
