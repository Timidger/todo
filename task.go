package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
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
	new, err := os.Create(save_path)
	if err != nil {
		panic(err)
	}
	new.WriteString(task.body_content)
}

/// Deletes a task by index
func DeleteTask(task_index int) {
	tasks := GetTasks()
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

// Determines if a task is due today (or any days before today)
func (task *Task) DueToday() bool {
	return task.due_date.Before(time.Now())
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
