package main

import (
	"errors"
	"fmt"
	"git.sr.ht/~sircmpwn/getopt"
	"git.sr.ht/~timidger/todo"
	"html/template"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)

const HELP_MESSAGE = "Usage of website:\n" +
	"  -p              Set the port number\n"
const WEBPAGE = "todo.html"

func main() {
	opts, _, err := getopt.Getopts(os.Args, "p:")
	if err != nil {
		fmt.Printf("%s", HELP_MESSAGE)
	}
	var port uint16
	port = 80

	for _, opt := range opts {
		switch opt.Option {
		case 'p':
			port_, err := strconv.ParseUint(opt.Value, 10, 16)
			if err != nil {
				todo.LogError(fmt.Sprintf("Bad port number: %v, %v", opt.Value, err))
				os.Exit(1)
			}
			port = uint16(port_)
		}
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", rootHandler)

	todo.LogError(fmt.Sprintf("%v\n", http.ListenAndServe(fmt.Sprintf(":%v", port), nil)))
	os.Exit(1)
}

type Result struct {
	Categories todo.Categories
	Tasks      todo.Tasks
}

func rootHandler(w http.ResponseWriter, req *http.Request) {
	var task_manager todo.TaskManager
	var cmd_manager todo.CommandManager

	task_manager.StorageDirectory = path.Join(os.Getenv("HOME"), ".todo/")
	cmd_manager.DueDate = time.Now()
	cmd_manager.Listing = todo.LISTING_DAY

	// The "/" pattern matches everything, so we need to check
	// that we're at the root here.
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	switch req.Method {
	case "DELETE":
		if err := req.ParseForm(); err != nil {
			fmt.Fprintf(w, "%v", err)
			return
		}
		category := req.FormValue("category")
		task_id := req.FormValue("task_id")
		err := delete_task(&task_manager, &cmd_manager, category, task_id)
		if err != nil {
			fmt.Fprintf(w, "%v\n", err)
		}
		// TODO Refresh page
	case "POST":
		if err := req.ParseForm(); err != nil {
			fmt.Fprintf(w, "%v", err)
			return
		}
		category := req.FormValue("category")
		task_body := req.FormValue("task_body")
		err := create_task(&task_manager, &cmd_manager, category, task_body)
		if err != nil {
			fmt.Fprintf(w, "%v\n", err)
			return
		}
		fallthrough
	case "GET":
		templ := template.New(WEBPAGE)
		templ = templ.Funcs(template.FuncMap{
			"Deref": func(s *string) string {
				if s != nil {
					return *s
				}
				return ""
			}})

		templ, err := templ.ParseFiles(WEBPAGE)
		if err != nil {
			panic(err)
		}

		tasks, err := cmd_manager.GetTasks(&task_manager)
		if err != nil {
			panic(err)
		}
		result := Result{
			Categories: task_manager.GetCategories(),
			Tasks:      tasks}
		err = templ.Execute(w, result)
		if err != nil {
			panic(err)
		}
	}
}

func create_task(task_manager *todo.TaskManager, cmd_manager *todo.CommandManager,
	category, task_body string) error {

	original := task_manager.StorageDirectory
	defer reset_category(task_manager, original)
	set_category(task_manager, category)

	err := cmd_manager.CreateTask(task_manager, task_body)
	if err != nil {
		return err
	}
	return nil
}

func delete_task(task_manager *todo.TaskManager, cmd_manager *todo.CommandManager,
	category, task_id string) error {

	original := task_manager.StorageDirectory
	defer reset_category(task_manager, original)
	set_category(task_manager, category)

	_, err := cmd_manager.DeleteTask(task_manager, task_id, true)
	if err != nil {
		return err
	}
	return nil
}

func set_category(task_manager *todo.TaskManager, category string) error {
	if category != "" {
		category_path := path.Join(task_manager.StorageDirectory, category)
		if _, err := os.Stat(category_path); os.IsNotExist(err) {
			msg := fmt.Sprintf("Category \"%s\" does not exist", category)
			todo.LogError(msg)
			return errors.New(msg)
		}
		task_manager.StorageDirectory = path.Join(
			task_manager.StorageDirectory,
			category)
	}
	return nil
}

func reset_category(task_manager *todo.TaskManager, original string) {
	task_manager.StorageDirectory = original
}
