package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"git.sr.ht/~sircmpwn/getopt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
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
			print_tasks(get_tasks())
		case 'd':
			to_delete, err := strconv.ParseInt(opt.Value, 10, 64)
			if err != nil {
				panic(err)
			}
			delete_task(int(to_delete))
		}
	}
	if len(opts) > 0 && !date_set {
		return
	}
	if len(os.Args) > 1 {
		input := strings.Join(os.Args[others+1:], " ")
		add_task(input, due_date)
	} else {
		reader := bufio.NewReader(os.Stdin)
		read_in_task(reader, due_date)
	}
}

func get_path() string {
	home := os.Getenv("HOME")
	root := path.Join(home, ".todo/")
	if _, err := os.Stat(root); os.IsNotExist(err) {
		if err = os.Mkdir(root, 0700); err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}
	return root
}

// Read a task in from a reader and pass it off to add_task.
func read_in_task(reader *bufio.Reader, due_date time.Time) {
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	text := string(bytes)
	add_task(text, due_date)
}

// Add a task with the given body.
func add_task(text string, due_date time.Time) {
	root := get_path()
	if !utf8.ValidString(text) {
		panic(fmt.Sprintf("Invalid UTF-8 string: %v", text))
	}
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return
	}
	// TODO Get a better name scheme
	sha := sha1.New()
	sha.Write([]byte(text))
	save_path := path.Join(root, fmt.Sprintf("%x", sha.Sum(nil))+".todo")
	if _, err := os.Stat(save_path); !os.IsNotExist(err) {
		fmt.Println("You have already made that a task")
		os.Exit(1)
	}
	var task Task
	task.body_content = text
	new, err := os.Create(save_path)
	if err != nil {
		panic(err)
	}
	new.WriteString(task.body_content)
}

/// Deletes a task by index
func delete_task(task_index int) {
	tasks := get_tasks()
	if task_index < 0 {
		fmt.Println("Index must be non-negative")
		os.Exit(1)
	}
	if task_index >= len(tasks) {
		fmt.Println("Index too large")
		os.Exit(1)
	}
	fmt.Printf("%d: %v\n", task_index, tasks[task_index].body_content)
	if err := os.Remove(tasks[task_index].file_name); err != nil {
		panic(err)
	}
	tasks = append(tasks[:task_index], tasks[task_index+1:]...)
}

// Print the tasks from ~/.todo
func print_tasks(tasks []Task) {
	for i, task := range tasks {
		if task.due_today() {
			fmt.Println(fmt.Sprintf("%d: %v", i, task.body_content))
		}
	}
}

func get_tasks() []Task {
	root := get_path()
	var tasks []Task
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if path == root {
			return nil
		}
		var task Task
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		task.body_content = string(content)
		task.file_name = path
		if !utf8.ValidString(task.body_content) {
			panic(fmt.Sprintf("Invalid UTF-8 string: %v", task.body_content))
		}
		tasks = append(tasks, task)
		return nil
	})
	if err != nil {
		panic(err)
	}
	return tasks
}
