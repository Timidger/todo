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
	taskIDueDate := tasks[i].DueDate.AddDate(0, 0, tasks[i].OverdueDays)
	taskJDueDate := tasks[j].DueDate.AddDate(0, 0, tasks[j].OverdueDays)
	return taskIDueDate.Before(taskJDueDate)
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
			truncatedName := string(task.index[0:length])
			if _, exists := hashes[truncatedName]; exists {
				length++
				hashes = make(map[string]*Task)
				continue condense_tasks
			}
			hashes[truncatedName] = task
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
		if tasks[i].index == hash || tasks[i].fullIndex[0:len(hash)] == hash {
			return &tasks[i]
		}
	}
	return nil
}

type TaskManager struct {
	StorageDirectory string
}

// Saves a new task to disk
func (manager *TaskManager) SaveTask(task *Task) error {
	storageDir := manager.StorageDirectory
	if task.category != nil && path.Base(storageDir) != *task.category {
		storageDir = path.Join(storageDir, *task.category)
	}
	createDir(storageDir)
	sha := sha1.New()
	sha.Write([]byte(task.BodyContent))
	// Also use the storage directory name as part of the hash,
	// this is to avoid collisions across categories.
	sha.Write([]byte(path.Base(storageDir)))
	hash := fmt.Sprintf("%x", sha.Sum(nil))
	task.fullIndex = hash
	savePath := path.Join(storageDir, hash+".todo")
	if _, err := os.Stat(savePath); !os.IsNotExist(err) {
		return errors.New("You have already made that a task")
	}
	new, err := os.Create(savePath)
	if err != nil {
		panic(err)
	}
	defer new.Close()

	taskJson, err := json.Marshal(task)
	if err != nil {
		return err
	}

	if _, err := new.Write(taskJson); err != nil {
		return err
	}
	return nil
}

/// Deletes a task by index
func (manager *TaskManager) DeleteTask(tasks Tasks, taskIndex string) *Task {
	toDeleteIndex := -1
	for i, task := range tasks {
		if taskIndex == task.index {
			toDeleteIndex = i
			break
		}
	}

	// If exact match couldn't be found see if there's a unique match using
	// the full length.
	if toDeleteIndex == -1 {
		for i, task := range tasks {
			if strings.HasPrefix(task.fullIndex, taskIndex) {
				if toDeleteIndex != -1 {
					return nil
				}
				toDeleteIndex = i
			}
		}
	}
	if toDeleteIndex == -1 {
		return nil
	}

	if err := os.Remove(tasks[toDeleteIndex].fileName); err != nil {
		panic(err)
	}
	task := tasks[toDeleteIndex]
	if task.Repeat == nil {
		tasks = append(tasks[:toDeleteIndex], tasks[toDeleteIndex+1:]...)
	}
	return &task
}

func (manager *TaskManager) GetCategories() Categories {
	createDir(manager.StorageDirectory)
	root := manager.StorageDirectory
	var categories Categories
	maxDepth := strings.Count(root, "/") + 1
	err := filepath.Walk(root,
		func(path string, info os.FileInfo, err error) error {
			depth := strings.Count(path, "/")
			if path == root || depth > maxDepth {
				return nil
			}
			if info.IsDir() {
				subDirFiles, err := ioutil.ReadDir(path)
				if err != nil {
					return err
				}
				count := 0
				for _, file := range subDirFiles {
					var task Task
					name := file.Name()
					if name == "audit_log" {
						continue
					}
					path := strings.Join([]string{path, name}, "/")
					bytes, err := ioutil.ReadFile(path)
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

func (manager *TaskManager) getTasksHelper() Tasks {
	createDir(manager.StorageDirectory)
	root := manager.StorageDirectory
	var tasks Tasks
	maxDepth := strings.Count(root, "/") + 1
	err := filepath.Walk(root,
		func(path string, info os.FileInfo, err error) error {
			// Nested directories are ignored -- this is to support categories
			// as simply "other" todo storages.
			depth := strings.Count(path, "/")
			if path == root || info.IsDir() || depth > maxDepth {
				return nil
			}
			var task Task
			task.fileName = path

			_, fileName := filepath.Split(path)
			split := strings.Split(fileName, ".")
			if len(split) < 2 || split[1] != "todo" {
				return nil
			}
			fileName = split[0]
			task.index = fileName
			task.fullIndex = fileName
			bytes, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			err = json.Unmarshal(bytes, &task)
			if err != nil {
				return err
			}
			task.index = fileName
			task.fullIndex = fileName
			tasks = append(tasks, task)
			return nil
		})
	if err != nil {
		panic(err)
	}
	return tasks
}

func (manager *TaskManager) GetTasks() Tasks {
	tasks := manager.getTasksHelper()
	categories := manager.GetCategories()
	originalDirectory := manager.StorageDirectory
	for _, category := range categories {
		manager.StorageDirectory = path.Join(originalDirectory, category.Name)
		categoryName := category.Name
		newTasks := manager.getTasksHelper()
		for i, _ := range newTasks {
			newTasks[i].category = &categoryName
		}
		tasks = append(tasks, newTasks...)
	}
	manager.StorageDirectory = originalDirectory
	tasks = tasks.Condense()
	sort.Sort(tasks)
	return tasks
}

func (tasks *Tasks) RemoveFirst(toRemove Task) {
	for i, task := range *tasks {
		if task == toRemove {
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

func (manager *TaskManager) AuditLog(task Task, done_date time.Time, annotation string) {
	createDir(manager.StorageDirectory)
	auditLogPath := path.Join(manager.StorageDirectory, AUDIT_LOG)
	var auditLogFile *os.File
	if _, err := os.Stat(auditLogPath); os.IsNotExist(err) {
		perms := os.O_APPEND | os.O_WRONLY | os.O_CREATE
		auditLogFile, err = os.OpenFile(auditLogPath, perms, 0600)
		if err != nil {
			panic(err)
		}
		auditLogFile.WriteString("#" + AUDIT_FIELDS)
	} else {
		auditLogFile, err = os.OpenFile(auditLogPath, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
	}

	defer auditLogFile.Close()

	audit_log := csv.NewWriter(auditLogFile)

	var record Record
	record.BodyContent = task.BodyContent
	record.DueDate = task.DueDate
	record.Repeat = task.Repeat
	record.OverdueDays = task.OverdueDays
	record.DateCompleted = done_date
	record.Annotation = annotation

	if err := audit_log.Write(record.Marshal()); err != nil {
		panic(err)
	}
	audit_log.Flush()
}

func (manager *TaskManager) AuditRecords() Records {
	createDir(manager.StorageDirectory)
	categories := manager.GetCategories()
	var records Records

	// Append nil category to get the root audit log
	for _, category := range append(categories, Category{}) {
		auditLogPath := path.Join(manager.StorageDirectory, category.Name, AUDIT_LOG)
		if _, err := os.Stat(auditLogPath); os.IsNotExist(err) {
			continue
		}

		auditLogFile, err := os.OpenFile(auditLogPath, os.O_RDONLY, 0600)
		if err != nil {
			panic(err)
		}
		defer auditLogFile.Close()

		audit_log := csv.NewReader(auditLogFile)
		audit_log.FieldsPerRecord = -1
		audit_log.Comment = '#'

		readRecords, err := audit_log.ReadAll()
		if err != nil {
			panic(err)
		}
		for _, readRecord := range readRecords {
			record := Unmarshal(readRecord)
			record.Category = category.Name
			records = append(records, record)
		}
	}

	sort.Sort(records)

	return records
}

// Creates a directory if it does not exist
func createDir(directoryPath string) {
	if _, err := os.Stat(directoryPath); os.IsNotExist(err) {
		if err = os.Mkdir(directoryPath, 0700); err != nil {
			panic(err)
		}
	} else if err != nil {
		msg := fmt.Sprintf("Could not read task storage: %v", err)
		LogError(msg)
		panic(msg)
	}
}
