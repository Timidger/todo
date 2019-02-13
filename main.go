package main // import "git.sr.ht/~timidger/todo"

import (
	"bufio"
	"crypto/sha1"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

func main() {
	var to_list = flag.Bool("list", false, "List the things to do in no particular order")
	flag.Parse()
	switch *to_list {
	case false:
		add_task()
	case true:
		print_tasks()
	}
}

/// Add a task by reading in from STDIN
func add_task() {
	// TODO duplicated
	home := os.Getenv("HOME")
	root := path.Join(home, ".todo/")
	if _, err := os.Stat(root); os.IsNotExist(err) {
		if err = os.Mkdir(root, 0700); err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}

	reader := bufio.NewReader(os.Stdin)
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	// TODO Check is utf8
	text := string(bytes)
	// TODO Get a better name scheme
	sha := sha1.New()
	sha.Write(bytes)
	save_path := path.Join(root, fmt.Sprintf("%x", sha.Sum(nil))+".todo")
	var task Task
	task.body_content = text
	// TODO asserts no exists
	new, err := os.Create(save_path)
	if err != nil {
		panic(err)
	}
	new.WriteString(task.body_content)
}

// Print the tasks from ~/.todo
func print_tasks() {
	home := os.Getenv("HOME")
	root := path.Join(home, ".todo/")
	if _, err := os.Stat(root); os.IsNotExist(err) {
		if err = os.Mkdir(root, 0700); err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}

	var todos []Task
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.Name() == ".todo" {
			return nil
		}
		var task Task
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		// TODO Check is utf8
		task.body_content = string(content)
		todos = append(todos, task)
		return nil
	})
	for _, todo := range todos {
		fmt.Println(todo.body_content)
	}
	if err != nil {
		panic(err)
	}
}
