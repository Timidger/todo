package main

import (
	"bufio"
	"fmt"
	"git.sr.ht/~sircmpwn/getopt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

const help_message = "Usage of todo:\n" +
	"  -h            Show this help message\n" +
	"  -l            List the things to do today in no particular order\n" +
	"  -d <value>    Delete a task by index number\n" +
	"  -t YYYY/MM/DD Delay the task until the date\n"

func main() {
	opts, others, err := getopt.Getopts(os.Args[1:], "hlt:d:")
	if err != nil {
		panic(err)
	}
	due_date := time.Now()
	date_set := false
	for _, opt := range opts {
		switch opt.Option {
		case 't':
			due_date, err = time.Parse("2006/01/02", opt.Value)
			if err != nil {
				panic(err)
			}
			date_set = true
		case 'h':
			fmt.Printf(help_message)
			return
		case 'l':
			printTasksToday(GetTasks())
		case 'd':
			to_delete, err := strconv.ParseInt(opt.Value, 10, 64)
			if err != nil {
				panic(err)
			}
			DeleteTask(int(to_delete))
		}
	}
	if len(opts) > 0 && !date_set {
		return
	}
	if len(os.Args) > 1 {
		input := strings.Join(os.Args[others+1:], " ")
		AddTask(input, due_date)
	} else {
		reader := bufio.NewReader(os.Stdin)
		readInTask(reader, due_date)
	}
}

// Read a task in from a reader and pass it off to add_task.
func readInTask(reader *bufio.Reader, due_date time.Time) {
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	text := string(bytes)
	AddTask(text, due_date)
}

// Print the tasks from ~/.todo to do today
func printTasksToday(tasks []Task) {
	for i, task := range tasks {
		if task.DueToday() {
			fmt.Println(fmt.Sprintf("%d: %v", i, task.body_content))
		}
	}
}
