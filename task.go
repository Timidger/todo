package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Task struct {
	// The "body" content of the task, in Markdown
	body_content string
	// When this task is due and must be done.
	// What this actually means is still up for debate.
	// However, I feel strongly that this should "explode" in some way.
	due_date time.Time
	// TODO REMOVE. This is a stupid hack.
	file_name string
}

// TODO Surely this an interface...
// Format a task
func (task *Task) FormatTask() string {
	format_string := "%-40v\t%v"
	if task.DueBeforeToday() {
		format_string = RED + format_string + RESET
	}
	return fmt.Sprintf(format_string,
		task.body_content, task.due_date.Format(TIME_FORMAT))
}

// Determines if a task is due today (or any days before today)
func (task *Task) DueToday() bool {
	return task.due_date.Before(time.Now())
}

/// Determines if a task is due before today.
func (task *Task) DueBeforeToday() bool {
	now := time.Now()
	day := string(strconv.Itoa(now.Day()))
	month := string(strconv.Itoa(int(now.Month())))
	if now.Day() < 10 {
		day = "0" + day
	}
	if now.Month() < 10 {
		month = "0" + month
	}
	today, err := time.Parse(TIME_FORMAT,
		fmt.Sprintf("%v/%v/%v EST", now.Year(), month, day))
	if err != nil {
		panic(err)
	}
	return task.due_date.Before(today)
}

// Adds a task with the given text and due time
func AddTask(text string, due_date time.Time) {
	root := getPath()
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
		// TODO proper error messages
		os.Exit(1)
	}
	var task Task
	task.body_content = text
	task.due_date = due_date
	new, err := os.Create(save_path)
	if err != nil {
		panic(err)
	}
	defer new.Close()
	if _, err := new.WriteString(fmt.Sprintf("%v\n", due_date.Format(TIME_FORMAT))); err != nil {
		panic(err)
	}
	if _, err := new.WriteString(task.body_content); err != nil {
		panic(err)
	}
}

/// Deletes a task by index
func DeleteTask(tasks []Task, task_index int) {
	if task_index < 0 {
		fmt.Println("Index must be non-negative")
		// TODO Do proper error handling
		os.Exit(1)
	}
	if task_index >= len(tasks) {
		fmt.Println("Index too large")
		// TODO Do proper error handling
		os.Exit(1)
	}
	fmt.Printf("%d: %v\n", task_index, tasks[task_index].body_content)
	if err := os.Remove(tasks[task_index].file_name); err != nil {
		panic(err)
	}
	tasks = append(tasks[:task_index], tasks[task_index+1:]...)
}

func GetTasks() []Task {
	root := getPath()
	var tasks []Task
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if path == root {
			return nil
		}
		var task Task
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		if !utf8.ValidString(string(bytes)) {
			panic(fmt.Sprintf("Invalid UTF-8 string: %v", bytes))
		}
		split := strings.SplitN(string(bytes), "\n", 2)
		if len(split) < 2 {
			panic(fmt.Sprintf("Invalid format \"%v\"", string(bytes)))
		}
		task.due_date, err = time.Parse(TIME_FORMAT, split[0])
		if err != nil {
			panic(err)
		}
		task.body_content = split[1]
		task.file_name = path
		tasks = append(tasks, task)
		return nil
	})
	if err != nil {
		panic(err)
	}
	return tasks
}

func getPath() string {
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
