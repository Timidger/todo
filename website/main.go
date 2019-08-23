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

var TASK_MANAGER todo.TaskManager
var CMD_MANAGER todo.CommandManager

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

	TASK_MANAGER.StorageDirectory = path.Join(os.Getenv("HOME"), ".todo/")
	CMD_MANAGER.DueDate = time.Now()
	CMD_MANAGER.Listing = todo.LISTING_DAY

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
	// The "/" pattern matches everything, so we need to check
	// that we're at the root here.
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	switch req.Method {
	case "POST":
		if err := req.ParseForm(); err != nil {
			fmt.Fprintf(w, "%v", err)
			return
		}
		category := req.FormValue("category")
		task_body := req.FormValue("task_body")
		err := create_task(category, task_body)
		if err != nil {
			w.Write(([]byte)(fmt.Sprintf("%v\n", err)))
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

		result := Result{
			Categories: TASK_MANAGER.GetCategories(),
			Tasks:      TASK_MANAGER.GetTasks()}
		err = templ.Execute(w, result)
		if err != nil {
			panic(err)
		}
	}
}

func create_task(category, task_body string) error {
	original := TASK_MANAGER.StorageDirectory
	if category != "" {
		category_path := path.Join(TASK_MANAGER.StorageDirectory, category)
		if _, err := os.Stat(category_path); os.IsNotExist(err) {
			msg := fmt.Sprintf("Category \"%s\" does not exist", category)
			todo.LogError(msg)
			return errors.New(msg)
		}
		TASK_MANAGER.StorageDirectory = path.Join(
			TASK_MANAGER.StorageDirectory,
			category)
	}
	err := CMD_MANAGER.CreateTask(&TASK_MANAGER, task_body)
	TASK_MANAGER.StorageDirectory = original
	if err != nil {
		return err
	}
	return nil
}
