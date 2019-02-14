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
	"unicode/utf8"
)

func main() {
	var to_list = flag.Bool("list", false, "List the things to do in no particular order")
	flag.Parse()
	switch *to_list {
	case false:
		add_task()
	case true:
		print_tasks(get_tasks())
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
func add_task() {
	root := get_path()
	reader := bufio.NewReader(os.Stdin)
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	text := string(bytes)
	if text == "" {
		return
	}
	if !utf8.ValidString(text) {
		panic(fmt.Sprintf("Invalid UTF-8 string: %v", text))
	}
	// TODO Get a better name scheme
	sha := sha1.New()
	sha.Write(bytes)
	save_path := path.Join(root, fmt.Sprintf("%x", sha.Sum(nil))+".todo")
	if _, err := os.Stat(save_path); !os.IsNotExist(err) {
		panic(fmt.Sprintf("You have already made that a task", save_path))
	}
	var task Task
	task.body_content = text
	new, err := os.Create(save_path)
	if err != nil {
		panic(err)
	}
	new.WriteString(task.body_content)
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
