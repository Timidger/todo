package main

import (
	"bufio"
	"crypto/sha1"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

func main() {
	var to_list = flag.Bool("l", false, "List the things to do in no particular order")
	var delete = flag.Int("d", -1, "Deletes a task by index number")
	flag.Parse()
	switch {
	case *delete >= 0:
		delete_task(*delete)
	case len(os.Args[1:]) >= 1 && !*to_list:
		add_task(strings.Join(os.Args[1:], " "))
	case *to_list:
		print_tasks(get_tasks())
	// default case
	case !*to_list:
		add_task("")
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

/// Add a task by reading in from STDIN
func add_task(text string) {
	root := get_path()
	if text == "" {
		reader := bufio.NewReader(os.Stdin)
		bytes, err := ioutil.ReadAll(reader)
		if err != nil {
			panic(err)
		}
		text = string(bytes)
	}
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
	if task_index >= len(tasks) {
		fmt.Println("Index too large")
		os.Exit(1)
	}
	if err := os.Remove(tasks[task_index].file_name); err != nil {
		panic(err)
	}
	tasks = append(tasks[:task_index], tasks[task_index+1:]...)
}

// Print the tasks from ~/.todo
func print_tasks(tasks []Task) {
	for i, task := range tasks {
		fmt.Println(fmt.Sprintf("%d: %v", i, task.body_content))
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
