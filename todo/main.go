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

const HELP_MESSAGE = "Usage of todo:\n" +
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
	"                  Default 0, Must be either a positive number or days separated by \",\".\n" +
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
		fmt.Printf("%s", HELP_MESSAGE)
		return
	}
	var task *todo.Task
	var taskManager todo.TaskManager
	taskManager.StorageDirectory = path.Join(os.Getenv("HOME"), ".todo/")

	var cmdManager todo.CommandManager
	cmdManager.DueDate = time.Now()
	cmdManager.Listing = todo.LISTING_DAY

	instantDelete := execute_flag_commands(&taskManager, &cmdManager, opts)

	if len(opts) == 0 || !cmdManager.SkipTaskCreationPrompt {
		input := strings.Join(os.Args[others:], " ")
		if len(os.Args) > 1 && input != "" {
			task, err = cmdManager.CreateTask(&taskManager, input)
		} else {
			reader := bufio.NewReader(os.Stdin)
			task, err = cmdManager.CreateTask(&taskManager, readInTask(reader))
		}
		if err != nil {
			todo.LogError(err.Error())
			os.Exit(1)
		}

		if instantDelete {
			todo.ClearCache()
			taskDeleted, err := cmdManager.DeleteTask(&taskManager, task.GetFullIndex(), false)
			if err != nil {
				todo.LogError(err.Error())
				os.Exit(1)
			}
			todo.LogSuccess(taskDeleted.String())
		}
	}

	// XXX At this point we can mess with the state as much as we want
	// We want all to remove all out of date tasks at this point, so we
	// the default state.
	cmdManager = todo.CommandManager{}
	cmdManager.DueDate = time.Now()
	cmdManager.Listing = todo.LISTING_DAY

	taskManager = todo.TaskManager{}
	taskManager.StorageDirectory = path.Join(os.Getenv("HOME"), ".todo/")

	tasks, err := cmdManager.GetTasks(&taskManager)
	if err != nil {
		todo.LogError(err.Error())
		os.Exit(1)
	}
	cmdManager.RemoveOverdueTasks(tasks, &taskManager)
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

func execute_flag_commands(taskManager *todo.TaskManager,
	cmdManager *todo.CommandManager, opts []getopt.Option) bool {
	instantDelete := false

	for _, opt := range opts {
		switch opt.Option {
		case 'h':
			fmt.Printf("%s", HELP_MESSAGE)
			os.Exit(0)
		case 't':
			zone, _ := time.Now().Zone()
			dueDate, err := time.Parse(todo.EXPLICIT_TIME_FORMAT,
				fmt.Sprintf("%v %s", opt.Value, zone))
			if err == nil {
				cmdManager.SetDueDate(dueDate)
			} else {
				err = cmdManager.SetDueDateRelative(opt.Value)
				if err != nil {
					todo.LogError(err.Error())
					os.Exit(1)
				}
			}
		case 'l':
			tasks, err := cmdManager.GetTasks(taskManager)
			if err != nil {
				todo.LogError(err.Error())
				os.Exit(1)
			}

			todo.DisplayTasks(tasks)
		case 'a':
			cmdManager.UseAllTasks()
		case 'D':
			taskDeleted, err := cmdManager.DeleteTask(taskManager, opt.Value, true)
			if err != nil {
				todo.LogError(err.Error())
				os.Exit(1)
			}
			if !isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Printf(taskDeleted.BodyContent)
			}
		case 'd':
			if opt.Value == "this" {
				instantDelete = true
				continue
			}
			taskDeleted, err := cmdManager.DeleteTask(taskManager, opt.Value, false)
			if err != nil {
				todo.LogError(err.Error())
				os.Exit(1)
			}
			if !isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Printf(taskDeleted.BodyContent)
			} else {
				todo.LogSuccess(taskDeleted.String())
			}
		case 's':
			err := cmdManager.SkipTask(taskManager, opt.Value)
			if err != nil {
				todo.LogError(err.Error())
				os.Exit(1)
			}
		case 'r':
			days, err := strconv.ParseInt(opt.Value, 10, 32)
			if err != nil {
				err = cmdManager.SetRepeatHumany(opt.Value)
			} else {
				err = cmdManager.SetRepeat(int(days))
			}
			if err != nil {
				todo.LogError(err.Error())
				os.Exit(1)
			}
		case 'x':
			err := cmdManager.DelayTask(taskManager, opt.Value)
			if err != nil {
				todo.LogError(err.Error())
				os.Exit(1)
			}
		case 'S':
			taskManager.StorageDirectory = opt.Value
		case 'c':
			category := opt.Value
			categoryPath := path.Join(taskManager.StorageDirectory, category)
			if _, err := os.Stat(categoryPath); os.IsNotExist(err) {
				todo.LogError(fmt.Sprintf("Category \"%s\" does not exist", category))
				os.Exit(1)
			}
			fallthrough
		case 'C':
			taskManager.StorageDirectory = path.Join(taskManager.StorageDirectory, opt.Value)
		case 'L':
			categories := cmdManager.GetCategories(taskManager)
			for _, category := range categories {
				fmt.Println(category)
			}
		case 'n':
			days, err := strconv.ParseInt(opt.Value, 10, 32)
			if err != nil {
				todo.LogError(fmt.Sprintf("Bad day delay \"%s\", need number", opt.Value))
				os.Exit(1)
			}

			err = cmdManager.SetDelay(int(days))
			if err != nil {
				todo.LogError(err.Error())
				os.Exit(1)
			}
		case 'A':
			records := cmdManager.GetAuditLog(taskManager)
			for _, record := range records {
				fmt.Println(record.String())
			}
		case 'e':
			cmdManager.Annotation = opt.Value
		}

	}

	tasksAll := cmdManager.GetTasksIfAll(taskManager)
	todo.DisplayTasksLong(tasksAll)

	return instantDelete
}
