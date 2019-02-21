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
	"  -a            List all the things to do, regardless of due date, in no particular order\n" +
	"  -d <value>    Delete a task by index number. If preceded by -a based on full list, not just today\n" +
	"  -t YYYY/MM/DD Delay the task until the date\n"

const TIME_FORMAT = "2006/01/02"

// TODO I'm trying to encode an enum but this feels gross
const (
	LISTING_ALL = iota
	LISTING_TODAY
)

func main() {
	opts, others, err := getopt.Getopts(os.Args[1:], "halt:d:")
	if err != nil {
		panic(err)
	}
	due_date := time.Now()
	date_set := false
	listing := LISTING_TODAY
	for _, opt := range opts {
		switch opt.Option {
		case 't':
			due_date, err = time.Parse(TIME_FORMAT, opt.Value)
			if err != nil {
				panic(err)
			}
			date_set = true
		case 'h':
			fmt.Printf(help_message)
			return
		case 'l':
			listing = LISTING_TODAY
			tasks := GetTasks()
			i := 0
			for _, task := range tasks {
				if task.DueToday() {
					fmt.Println(fmt.Sprintf("%d: %v", i, task.body_content))
					i++
				}
			}
		case 'a':
			listing = LISTING_ALL
			tasks := GetTasks()
			for i, task := range tasks {
				fmt.Println(fmt.Sprintf("%d: %v", i, task.body_content))
			}
		case 'd':
			to_delete, err := strconv.ParseInt(opt.Value, 10, 64)
			if err != nil {
				panic(err)
			}
			switch listing {
			case LISTING_ALL:
				tasks := GetTasks()
				DeleteTask(tasks, int(to_delete))
			case LISTING_TODAY:
				tasks_ := GetTasks()
				tasks := make([]Task, 0)
				for _, task := range tasks_ {
					if task.DueToday() {
						tasks = append(tasks, task)
					}
				}
				DeleteTask(tasks, int(to_delete))
			default:
				panic(fmt.Sprintf("Unknown flag %v", listing))
			}
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
