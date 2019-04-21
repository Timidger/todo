package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type Tasks []Task

func (tasks Tasks) Len() int {
	return len(tasks)
}

func (tasks Tasks) Less(i, j int) bool {
	return tasks[i].due_date.Before(tasks[j].due_date)
}

func (tasks Tasks) Swap(i, j int) {
	tasks[i], tasks[j] = tasks[j], tasks[i]
}

type TaskManager struct {
	storage_directory string
}

func (manager *TaskManager) AddTask(text string, due_date time.Time) {
	create_dir(manager.storage_directory)
	root := manager.storage_directory
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
	if _, err := new.WriteString(fmt.Sprintf("%v\n", due_date.Format(EXPLICIT_TIME_FORMAT))); err != nil {
		panic(err)
	}
	if _, err := new.WriteString(task.body_content); err != nil {
		panic(err)
	}
}

/// Deletes a task by index
func (manager *TaskManager) DeleteTask(tasks Tasks, task_index int) *Task {
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
	if err := os.Remove(tasks[task_index].file_name); err != nil {
		panic(err)
	}
	task := tasks[task_index]
	tasks = append(tasks[:task_index], tasks[task_index+1:]...)
	return &task
}

func (manager *TaskManager) GetTasks() Tasks {
	create_dir(manager.storage_directory)
	root := manager.storage_directory
	var tasks Tasks
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
		task.due_date, err = time.Parse(EXPLICIT_TIME_FORMAT, split[0])
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
	sort.Sort(tasks)
	return tasks
}

func (manager *TaskManager) GetTasksToday() []Task {
	tasks_ := manager.GetTasks()
	tasks := make([]Task, 0)
	for _, task := range tasks_ {
		if task.DueToday() {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// Creates a directory if it does not exist
func create_dir(directory_path string) {
	if _, err := os.Stat(directory_path); os.IsNotExist(err) {
		if err = os.Mkdir(directory_path, 0700); err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}
}
