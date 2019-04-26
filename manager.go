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
	if tasks[i].due_date == nil || tasks[j].due_date == nil {
		return false
	}
	return tasks[i].due_date.Before(*tasks[j].due_date)
}

func (tasks Tasks) Swap(i, j int) {
	tasks[i], tasks[j] = tasks[j], tasks[i]
}

/// Condenses tasks to be the smallest possible hash value possible
func (tasks Tasks) Condense() Tasks {
	hashes := make(map[string]*Task)
	length := 1
condense_tasks:
	for {
		for i, _ := range tasks {
			task := &tasks[i]
			truncated_name := string(task.index[0:length])
			if _, exists := hashes[truncated_name]; exists {
				length++
				hashes = make(map[string]*Task)
				continue condense_tasks
			}
			hashes[truncated_name] = task
		}
		break
	}
	// Go (heh) through and clean up the indices to be as minimal as possible
	for hash, task := range hashes {
		task.index = hash
	}
	return tasks
}

type TaskManager struct {
	storage_directory string
}

func (manager *TaskManager) AddTask(text string, due_date *time.Time) *Task {
	create_dir(manager.storage_directory)
	root := manager.storage_directory
	if !utf8.ValidString(text) {
		panic(fmt.Sprintf("Invalid UTF-8 string: %v", text))
	}
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return nil
	}
	// TODO Get a better name scheme
	sha := sha1.New()
	sha.Write([]byte(text))
	// Also use the storage directory name as part of the hash,
	// this is to avoid collisions across categories.
	sha.Write([]byte(path.Base(manager.storage_directory)))
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
	if due_date != nil {
		if _, err := new.WriteString(fmt.Sprintf("%v\n", due_date.Format(EXPLICIT_TIME_FORMAT))); err != nil {
			panic(err)
		}
	}
	if _, err := new.WriteString(task.body_content); err != nil {
		panic(err)
	}
	return &task
}

/// Deletes a task by index
func (manager *TaskManager) DeleteTask(tasks Tasks, task_index string) *Task {
	for i, task := range tasks {
		if task.index == task_index {
			if err := os.Remove(tasks[i].file_name); err != nil {
				panic(err)
			}
			tasks = append(tasks[:i], tasks[i+1:]...)
			return &task
		}
	}
	return nil
}

func (manager *TaskManager) GetCategories() Categories {
	create_dir(manager.storage_directory)
	root := manager.storage_directory
	var categories Categories
	max_depth := strings.Count(root, "/") + 1
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		depth := strings.Count(path, "/")
		if path == root || depth > max_depth {
			return nil
		}
		if info.IsDir() {
			sub_dir_files, err := ioutil.ReadDir(path)
			if err != nil {
				return err
			}
			var category Category
			category.Name = info.Name()
			category.Tasks = len(sub_dir_files)
			categories = append(categories, category)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	sort.Sort(categories)
	return categories
}

func (manager *TaskManager) get_tasks_helper() Tasks {
	create_dir(manager.storage_directory)
	root := manager.storage_directory
	var tasks Tasks
	max_depth := strings.Count(root, "/") + 1
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		// Nested directories are ignored -- this is to support categories
		// as simply "other" todo storages.
		depth := strings.Count(path, "/")
		if path == root || info.IsDir() || depth > max_depth {
			return nil
		}
		var task Task
		task.file_name = path

		_, file_name := filepath.Split(path)
		file_name = strings.Split(file_name, ".")[0]
		task.index = file_name
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		if !utf8.ValidString(string(bytes)) {
			panic(fmt.Sprintf("Invalid UTF-8 string: %v", bytes))
		}
		split := strings.SplitN(string(bytes), "\n", 2)
		due_date, err := time.Parse(EXPLICIT_TIME_FORMAT, split[0])
		if err != nil {
			task.body_content = string(bytes)
		} else {
			task.due_date = &due_date
			task.body_content = split[1]
		}
		tasks = append(tasks, task)
		return nil
	})
	if err != nil {
		panic(err)
	}
	sort.Sort(tasks)
	return tasks
}

func (manager *TaskManager) GetTasks() Tasks {
	tasks := manager.get_tasks_helper()
	categories := manager.GetCategories()
	original_directory := manager.storage_directory
	for _, category := range categories {
		manager.storage_directory = path.Join(original_directory, category.Name)
		category_name := category.Name
		new_tasks := manager.get_tasks_helper()
		for i, _ := range new_tasks {
			new_tasks[i].category = &category_name
		}
		tasks = append(tasks, new_tasks...)
	}
	manager.storage_directory = original_directory
	tasks = tasks.Condense()
	return tasks
}

func (manager *TaskManager) GetTasksToday() []Task {
	tasks_ := manager.GetTasks()
	tasks := make(Tasks, 0)
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
