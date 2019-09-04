package todo

import (
	"crypto/sha1"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Tasks []Task

func (tasks Tasks) Len() int {
	return len(tasks)
}

func (tasks Tasks) Less(i, j int) bool {
	if tasks[i].category != nil {
		if tasks[j].category == nil {
			return false
		}
		if tasks[j].category != nil {
			comparison := strings.Compare(*tasks[i].category, *tasks[j].category)
			if comparison != 0 {
				return comparison < 0
			}
		}
	} else if tasks[j].category != nil {
		return true
	}
	task_i_due_date := tasks[i].Due_date.AddDate(0, 0, tasks[i].Overdue_days)
	task_j_due_date := tasks[j].Due_date.AddDate(0, 0, tasks[j].Overdue_days)
	return task_i_due_date.Before(task_j_due_date)
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

func (tasks Tasks) GetByHash(hash string) *Task {
	for i, _ := range tasks {
		if tasks[i].index == hash || tasks[i].full_index[0:len(hash)] == hash {
			return &tasks[i]
		}
	}
	return nil
}

type TaskManager struct {
	StorageDirectory string
}

// Saves a new task to disk
func (manager *TaskManager) SaveTask(task Task) error {
	storage_dir := manager.StorageDirectory
	if task.category != nil && path.Base(storage_dir) != *task.category {
		storage_dir = path.Join(storage_dir, *task.category)
	}
	create_dir(storage_dir)
	sha := sha1.New()
	sha.Write([]byte(task.Body_content))
	// Also use the storage directory name as part of the hash,
	// this is to avoid collisions across categories.
	sha.Write([]byte(path.Base(storage_dir)))
	save_path := path.Join(storage_dir, fmt.Sprintf("%x", sha.Sum(nil))+".todo")
	if _, err := os.Stat(save_path); !os.IsNotExist(err) {
		return errors.New("You have already made that a task")
	}
	new, err := os.Create(save_path)
	if err != nil {
		panic(err)
	}
	defer new.Close()

	task_json, err := json.Marshal(task)
	if err != nil {
		return err
	}

	if _, err := new.Write(task_json); err != nil {
		return err
	}
	return nil
}

/// Deletes a task by index
func (manager *TaskManager) DeleteTask(tasks Tasks, task_index string) *Task {
	to_delete_index := -1
	for i, task := range tasks {
		if task_index == task.index {
			to_delete_index = i
			break
		}
	}

	// If exact match couldn't be found see if there's a unique match using
	// the full length.
	if to_delete_index == -1 {
		for i, task := range tasks {
			if task_index == task.full_index[0:len(task_index)] {
				if to_delete_index != -1 {
					return nil
				}
				to_delete_index = i
			}
		}
	}
	if to_delete_index == -1 {
		return nil
	}

	if err := os.Remove(tasks[to_delete_index].file_name); err != nil {
		panic(err)
	}
	task := tasks[to_delete_index]
	if task.Repeat == nil {
		tasks = append(tasks[:to_delete_index], tasks[to_delete_index+1:]...)
	}
	return &task
}

func (manager *TaskManager) GetCategories() Categories {
	create_dir(manager.StorageDirectory)
	root := manager.StorageDirectory
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
			count := 0
			for _, file := range sub_dir_files {
				var task Task
				name := file.Name()
				if name == "audit_log" {
					continue
				}
				bytes, err := ioutil.ReadFile(strings.Join([]string{path, name}, "/"))
				if err != nil {
					continue
				}
				err = json.Unmarshal(bytes, &task)
				if err != nil {
					continue
				}
				if task.DueToday() {
					count += 1
				}
			}
			var category Category
			category.Name = info.Name()
			category.Tasks = count
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
	create_dir(manager.StorageDirectory)
	root := manager.StorageDirectory
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
		split := strings.Split(file_name, ".")
		if len(split) < 2 || split[1] != "todo" {
			return nil
		}
		file_name = split[0]
		task.index = file_name
		task.full_index = file_name
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		err = json.Unmarshal(bytes, &task)
		if err != nil {
			return err
		}
		task.index = file_name
		task.full_index = file_name
		tasks = append(tasks, task)
		return nil
	})
	if err != nil {
		panic(err)
	}
	return tasks
}

func (manager *TaskManager) GetTasks() Tasks {
	tasks := manager.get_tasks_helper()
	categories := manager.GetCategories()
	original_directory := manager.StorageDirectory
	for _, category := range categories {
		manager.StorageDirectory = path.Join(original_directory, category.Name)
		category_name := category.Name
		new_tasks := manager.get_tasks_helper()
		for i, _ := range new_tasks {
			new_tasks[i].category = &category_name
		}
		tasks = append(tasks, new_tasks...)
	}
	manager.StorageDirectory = original_directory
	tasks = tasks.Condense()
	sort.Sort(tasks)
	return tasks
}

func (tasks *Tasks) RemoveFirst(to_remove Task) {
	for i, task := range *tasks {
		if task == to_remove {
			*tasks = append((*tasks)[:i], (*tasks)[i+1:]...)
			return
		}
	}
}

func (tasks_ Tasks) FilterTasksDueOnDay(date time.Time) []Task {
	tasks := make(Tasks, 0)
	for _, task := range tasks_ {
		if task.DueOn(date) {
			tasks = append(tasks, task)
		}
	}
	sort.Sort(tasks)
	return tasks
}

func (tasks_ Tasks) FilterTasksDueBeforeToday() []Task {
	tasks := make(Tasks, 0)
	for _, task := range tasks_ {
		if task.DueToday() {
			tasks = append(tasks, task)
		}
	}
	sort.Sort(tasks)
	return tasks
}

func (manager *TaskManager) AuditLog(task Task, done_date time.Time) {
	create_dir(manager.StorageDirectory)
	audit_log_path := path.Join(manager.StorageDirectory, AUDIT_LOG)
	var audit_log_file *os.File
	if _, err := os.Stat(audit_log_path); os.IsNotExist(err) {
		audit_log_file, err = os.OpenFile(audit_log_path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}
		audit_log_file.WriteString("#" + Fields())
	} else {
		audit_log_file, err = os.OpenFile(audit_log_path, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
	}

	defer audit_log_file.Close()

	audit_log := csv.NewWriter(audit_log_file)

	var record Record
	record.Body_content = task.Body_content
	record.Due_date = task.Due_date
	record.Repeat = task.Repeat
	record.Overdue_days = task.Overdue_days
	record.DateCompleted = done_date

	if err := audit_log.Write(record.Marshal()); err != nil {
		panic(err)
	}
	audit_log.Flush()
}

func (manager *TaskManager) AuditRecords() Records {
	create_dir(manager.StorageDirectory)
	categories := manager.GetCategories()
	var records Records

	// Append nil category to get the root audit log
	for _, category := range append(categories, Category{}) {
		audit_log_path := path.Join(manager.StorageDirectory, category.Name, AUDIT_LOG)
		if _, err := os.Stat(audit_log_path); os.IsNotExist(err) {
			continue
		}

		audit_log_file, err := os.OpenFile(audit_log_path, os.O_RDONLY, 0600)
		if err != nil {
			panic(err)
		}
		defer audit_log_file.Close()

		audit_log := csv.NewReader(audit_log_file)
		audit_log.FieldsPerRecord = FieldCount()
		audit_log.Comment = '#'

		read_records, err := audit_log.ReadAll()
		if err != nil {
			panic(err)
		}
		for _, read_record := range read_records {
			record := Unmarshal(read_record)
			record.Category = category.Name
			records = append(records, record)
		}
	}

	sort.Sort(records)

	return records
}

// Creates a directory if it does not exist
func create_dir(directory_path string) {
	if _, err := os.Stat(directory_path); os.IsNotExist(err) {
		if err = os.Mkdir(directory_path, 0700); err != nil {
			panic(err)
		}
	} else if err != nil {
		msg := fmt.Sprintf("Could not read task storage: %v", err)
		LogError(msg)
		panic(msg)
	}
}
