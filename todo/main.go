package main

import (
	"bufio"
	"fmt"
	"git.sr.ht/~sircmpwn/getopt"
	"git.sr.ht/~timidger/todo"
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
	"  -d <index>      Delete a task by index number. If preceded by -a based on full list, not just due now.\n" +
	"                  \"this\" is a special case that will delete a task you create in the same command invocation.\n" +
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
	"  -e <notes>      Annotate task completion so it shows up in the audit log later\n" +
	"                  Can be paired with -d or -s\n" +
	"  -c <category>   Specify a category\n" +
	"  -C <category>   Create a new category\n" +
	"  -L              List all the categories\n" +
	"  -A              Show audit logs. Can be controlled with -t and -c\n" +
	"  -S <directory>  Specify a custom todo directory (default is ~/.todo). Primarily used for testing\n"

func main() {
	opts, others, err := getopt.Getopts(os.Args, "ALhalt:d:x:D:S:C:c:r:n:s:e:")
	if err != nil {
		fmt.Printf("%s", help_message)
		return
	}
	var task_manager todo.TaskManager
	task_manager.StorageDirectory = path.Join(os.Getenv("HOME"), ".todo/")

	var cmd_manager todo.CommandManager
	cmd_manager.DueDate = time.Now()
	cmd_manager.Listing = todo.LISTING_DAY

	instant_delete := false

	for _, opt := range opts {
		switch opt.Option {
		case 'h':
			fmt.Printf("%s", help_message)
			return
		case 't':
			due_date, err := time.Parse(todo.EXPLICIT_TIME_FORMAT, opt.Value+" EDT")
			if err == nil {
				cmd_manager.SetDueDate(due_date)
			} else {
				err = cmd_manager.SetDueDateRelative(opt.Value)
				if err != nil {
					todo.LogError(err.Error())
					os.Exit(1)
				}
			}
		case 'l':
			tasks, err := cmd_manager.GetTasks(&task_manager)
			if err != nil {
				todo.LogError(err.Error())
				os.Exit(1)
			}

			todo.DisplayTasks(tasks)
		case 'a':
			cmd_manager.UseAllTasks()
		case 'D':
			task_deleted, err := cmd_manager.DeleteTask(&task_manager, opt.Value, true)
			if err != nil {
				todo.LogError(err.Error())
				os.Exit(1)
			}
			if !isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Printf(task_deleted.Body_content)
			} else {
				todo.LogSuccess(task_deleted.String())
			}
		case 'd':
			if opt.Value == "this" {
				instant_delete = true
				continue
			}
			task_deleted, err := cmd_manager.DeleteTask(&task_manager, opt.Value, false)
			if err != nil {
				todo.LogError(err.Error())
				os.Exit(1)
			}
			if !isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Printf(task_deleted.Body_content)
			} else {
				todo.LogSuccess(task_deleted.String())
			}
		case 's':
			err := cmd_manager.SkipTask(&task_manager, opt.Value)
			if err != nil {
				todo.LogError(err.Error())
				os.Exit(1)
			}
		case 'r':
			days, err := strconv.ParseInt(opt.Value, 10, 32)
			if err != nil {
				err = cmd_manager.SetRepeatHumany(opt.Value)
			} else {
				err = cmd_manager.SetRepeat(int(days))
			}
			if err != nil {
				todo.LogError(err.Error())
				os.Exit(1)
			}
		case 'x':
			err := cmd_manager.DelayTask(&task_manager, opt.Value)
			if err != nil {
				todo.LogError(err.Error())
				os.Exit(1)
			}
		case 'S':
			task_manager.StorageDirectory = opt.Value
		case 'c':
			category := opt.Value
			category_path := path.Join(task_manager.StorageDirectory, category)
			if _, err := os.Stat(category_path); os.IsNotExist(err) {
				todo.LogError(fmt.Sprintf("Category \"%s\" does not exist", category))
				os.Exit(1)
			}
			fallthrough
		case 'C':
			task_manager.StorageDirectory = path.Join(task_manager.StorageDirectory, opt.Value)
		case 'L':
			categories := cmd_manager.GetCategories(&task_manager)
			for _, category := range categories {
				fmt.Println(category)
			}
		case 'n':
			days, err := strconv.ParseInt(opt.Value, 10, 32)
			if err != nil {
				todo.LogError(fmt.Sprintf("Bad day delay \"%s\", need number", opt.Value))
				os.Exit(1)
			}

			err = cmd_manager.SetDelay(int(days))
			if err != nil {
				todo.LogError(err.Error())
				os.Exit(1)
			}
		case 'A':
			records := cmd_manager.GetAuditLog(&task_manager)
			for _, record := range records {
				fmt.Println(record.String())
			}
		case 'e':
			cmd_manager.Annotation = opt.Value
		}

	}

	// if skip_task_read is set then we have done an action that means we should
	// not print this to stdout due to the chaining rule described above.
	tasks_all := cmd_manager.GetTasksIfAll(&task_manager)
	todo.DisplayTasksLong(tasks_all)

	if len(opts) > 0 && cmd_manager.SkipTaskCreationPrompt {
		return
	}

	var task *todo.Task
	if input := strings.Join(os.Args[others:], " "); len(os.Args) > 1 && input != "" {
		task, err = cmd_manager.CreateTask(&task_manager, input)
	} else {
		reader := bufio.NewReader(os.Stdin)
		task, err = cmd_manager.CreateTask(&task_manager, readInTask(reader))
	}
	if err != nil {
		todo.LogError(err.Error())
		os.Exit(1)
	}

	if instant_delete {
		todo.ClearCache()
		task_deleted, err := cmd_manager.DeleteTask(&task_manager, task.GetFullIndex(), false)
		if err != nil {
			todo.LogError(err.Error())
			os.Exit(1)
		}
		todo.LogSuccess(task_deleted.String())
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
