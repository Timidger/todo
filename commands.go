package todo

import (
	"errors"
	"fmt"
	"math"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	LISTING_ALL = iota
	LISTING_DAY
)

// This is done to reduce the number of read/writes for multiple operations
var tasks *Tasks = nil

func GetTasks(manager *TaskManager) *Tasks {
	if tasks == nil {
		tasks_ := manager.GetTasks()
		tasks = &tasks_
	}
	return tasks
}

func ClearCache() {
	tasks = nil
}

// Manages the state between commands
type CommandManager struct {
	// See LISTING enum
	Listing     int
	OverdueDays int
	DueDate     time.Time
	// If time is set manually we can behave differently
	TimeSet bool
	Repeat  *string
	// If certain actions have been taken skip task creation from stdin
	// This only makes sense for the command line.
	SkipTaskCreationPrompt bool
	// Annotation to add to the audit log when deleting a task.
	Annotation string
}

// -t
func (cmdManager *CommandManager) SetDueDate(newDueDate time.Time) {
	cmdManager.DueDate = newDueDate
	cmdManager.TimeSet = true
}

func getHumanyNextDelay(daysString string) (time.Time, error) {
	// Must be human-y
	today := time.Now()
	days := strings.Split(daysString, ",")
	if len(days) == 0 {
		return today, errors.New("Empty repeat is not allowed")
	}
	/* We want the lowest index day _after_ today's
	where Sunday = 0, Saturday = 6
	If there is no number after today then just the smallest one. */
	todayIndex := int(today.Weekday())
	dayIndices := make([]int, len(days))
	for index, day := range days {
		var err error
		dayIndices[index], err = dayToIndex(day)
		if err != nil {
			return today, err
		}
	}
	indexOfLowest := 0
	for index, dayIndex := range dayIndices[1:] {
		if dayIndices[indexOfLowest] < todayIndex && dayIndex < todayIndex {
			indexOfLowest = index
		}
		if dayIndex > todayIndex && dayIndex < dayIndices[indexOfLowest] {
			indexOfLowest = index
		}
	}
	if indexOfLowest == todayIndex {
		return today.AddDate(0, 0, 7), nil
	}
	return humanyTime(days[indexOfLowest])
}

func dayToIndex(day string) (int, error) {
	switch strings.Title(day) {
	case "Sunday":
		return 0, nil
	case "Monday":
		return 1, nil
	case "Tuesday":
		return 2, nil
	case "Wednesday":
		return 3, nil
	case "Thursday":
		return 4, nil
	case "Friday":
		return 5, nil
	case "Saturday":
		return 6, nil
	default:
		return 0, errors.New(fmt.Sprintf("Invalid day %s", day))
	}
}

func humanyTime(day string) (time.Time, error) {
	today := time.Now()
	relativeDay := 0
	curWeekday := int(today.Weekday())
	switch strings.Title(day) {
	case "Sunday":
		relativeDay = 0
	case "Monday":
		relativeDay = 1
	case "Tuesday":
		relativeDay = 2
	case "Wednesday":
		relativeDay = 3
	case "Thursday":
		relativeDay = 4
	case "Friday":
		relativeDay = 5
	case "Saturday":
		relativeDay = 6
	case "Tomorrow":
		relativeDay = (curWeekday + 1) % 7
	case "Today":
		return today, nil
	case "Yesterday":
		return today.AddDate(0, 0, -1), nil
	default:
		return today, errors.New(fmt.Sprintf("Invalid human-y day %s", day))
	}
	if curWeekday < relativeDay {
		return today.AddDate(0, 0, int(relativeDay-curWeekday)), nil
	} else {
		return today.AddDate(0, 0, 7-(curWeekday-relativeDay)), nil
	}
}

// -t
func (cmdManager *CommandManager) SetDueDateRelative(newDueDate string) error {
	cmdManager.TimeSet = true
	humanDueDate, err := humanyTime(newDueDate)
	if err != nil {
		cmdManager.DueDate = humanDueDate
		return nil
	}
	// Attempt to use date as a fallback, using ISO-8601 format
	var dueDate time.Time
	out, err := exec.Command("/usr/bin/date", "--iso-8601", "-d", newDueDate).Output()
	if err != nil {
		return errors.New(fmt.Sprintf("/usr/bin/date failed: %v", err))
	}
	newDueDate = strings.ReplaceAll(strings.TrimSpace(string(out)), "-", "/")
	zone, _ := time.Now().Zone()
	dueDate, err = time.Parse(EXPLICIT_TIME_FORMAT, fmt.Sprintf("%s %s", newDueDate, zone))
	if err != nil {
		return errors.New(fmt.Sprintf("Bad date: %s", newDueDate))
	}
	cmdManager.DueDate = dueDate
	return nil
}

// -l
func (cmdManager *CommandManager) GetTasks(taskManager *TaskManager) (Tasks, error) {
	allTasks := GetTasks(taskManager)
	cmdManager.Listing = LISTING_DAY
	cmdManager.SkipTaskCreationPrompt = true

	var tasks Tasks
	if cmdManager.DueDate.Before(time.Now()) && !cmdManager.TimeSet {
		tasks = allTasks.FilterTasksDueBeforeToday()
		if len(tasks) == 0 {
			tasks = *allTasks
		}
	} else {
		tasks = allTasks.FilterTasksDueOnDay(cmdManager.DueDate)
	}

	return tasks, nil
}

// What -a should be, don't list until we know we aren't gonna need to pipe
func (cmdManager *CommandManager) UseAllTasks() {
	cmdManager.Listing = LISTING_ALL
}

// -s
func (cmdManager *CommandManager) SkipTask(taskManager *TaskManager, index string) error {
	skip_task := GetTasks(taskManager).GetByHash(index)
	if skip_task.Repeat == nil {
		return errors.New("Can only skip repeat tasks")
	}
	_, err := cmdManager.deleteTaskHelper(taskManager, index, false, true)
	return err
}

// -D (true) and -d (false)
func (cmdManager *CommandManager) DeleteTask(taskManager *TaskManager, index string,
	force_delete bool) (*Task, error) {

	task, err := cmdManager.deleteTaskHelper(taskManager, index, force_delete, false)
	if err == nil && task == nil {
		panic("At least one value was expected to be non-nil")
	}
	return task, err
}

// helper for -D, -d, and -s
func (cmdManager *CommandManager) deleteTaskHelper(taskManager *TaskManager, index string,
	force_delete, skip_repeat bool) (*Task, error) {

	if force_delete && skip_repeat {
		panic("force_delete and skip_repeat cannot both be true")
	}

	allTasks := GetTasks(taskManager)
	cmdManager.SkipTaskCreationPrompt = true
	var taskDeleted *Task

	switch cmdManager.Listing {
	case LISTING_DAY:
		var tasks Tasks
		if cmdManager.DueDate.Before(time.Now()) {
			// NOTE This is a special case: we want everything due today
			// or before today with this call..
			tasks = allTasks.FilterTasksDueBeforeToday()
		} else {
			tasks = allTasks.FilterTasksDueOnDay(cmdManager.DueDate)
		}
		if len(tasks) != 0 {
			taskDeleted = taskManager.DeleteTask(tasks, index)
			if taskDeleted != nil && taskDeleted.Repeat == nil {
				allTasks.RemoveFirst(*taskDeleted)
			}
			break
		}
		// If there are no tasks today then we must try to delete based
		// on all tasks. This lets you use it like -l when there are no
		// tasks today.
		fallthrough
	case LISTING_ALL:
		taskDeleted = (taskManager).DeleteTask(*allTasks, index)
		if taskDeleted != nil {
			allTasks.RemoveFirst(*taskDeleted)
		}
	}

	if taskDeleted == nil {
		return nil, errors.New(fmt.Sprintf("Bad index \"%s\"", index))
	}

	if !force_delete && taskDeleted.Repeat != nil {
		// Recreate the task if it has a repeat.
		delay, err := strconv.Atoi(*taskDeleted.Repeat)
		if err != nil {
			taskDeleted.DueDate, err = getHumanyNextDelay(*taskDeleted.Repeat)
			if err != nil {
				return nil, err
			}
		} else {
			taskDeleted.DueDate = taskDeleted.DueDate.AddDate(0, 0, delay)
		}
		if err := taskManager.SaveTask(taskDeleted); err != nil {
			return nil, err
		}
	}

	// Log in the audit log
	if !force_delete && !skip_repeat {
		original_StorageDirectory := taskManager.StorageDirectory
		if taskDeleted.category != nil {
			taskManager.StorageDirectory = path.Join(taskManager.StorageDirectory, *taskDeleted.category)
		}
		taskManager.AuditLog(*taskDeleted, cmdManager.DueDate, cmdManager.Annotation)
		taskManager.StorageDirectory = original_StorageDirectory
	}

	return taskDeleted, nil
}

// Remove tasks that are overdue by 3 days.
func (cmdManager *CommandManager) RemoveOverdueTasks(tasks Tasks, taskManager *TaskManager) {
	for _, task := range tasks {
		finalDueDate := task.DueDate.AddDate(0, 0, task.OverdueDays)
		overdue_days := int(math.Floor(time.Now().Sub(finalDueDate).Hours() / 24))
		if overdue_days > 3 {
			// TODO Remove once I'm used to this feature
			LogError(fmt.Sprintf("Auto removing overdue task \"%v\"",
				task.BodyContent))
			if task.Repeat == nil {
				cmdManager.deleteTaskHelper(taskManager, task.fullIndex, true, false)
			} else {
				cmdManager.deleteTaskHelper(taskManager, task.fullIndex, false, true)
			}
		}
	}
}

// -r
func (cmdManager *CommandManager) SetRepeatHumany(days string) error {
	for _, day := range strings.Split(days, ",") {
		switch strings.Title(day) {
		case "Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday":
			continue
		default:
			return errors.New(fmt.Sprintf("Invalid repeat day %s", day))
		}
	}
	cmdManager.Repeat = &days
	return nil
}

// -r
func (cmdManager *CommandManager) SetRepeat(days int) error {
	if days <= 0 {
		return errors.New("Repeat time must be a positive, non-zero number")
	}

	hours := strconv.Itoa(int(days))
	cmdManager.Repeat = &hours

	return nil
}

// -n
func (cmdManager *CommandManager) SetDelay(days int) error {
	if days <= 0 {
		return errors.New("Delay time must be a positive, non-zero number")
	}
	cmdManager.OverdueDays = int(days)
	return nil
}

// -x
func (cmdManager *CommandManager) DelayTask(taskManager *TaskManager, index string) error {
	cmdManager.SkipTaskCreationPrompt = true
	// Always do a re-read for delays. Makes them both more expensive and multiple delays work.
	tasks = nil
	allTasks := GetTasks(taskManager)

	var tasks Tasks
	switch cmdManager.Listing {
	case LISTING_ALL:
		tasks = *allTasks
	case LISTING_DAY:
		tasks = allTasks.FilterTasksDueBeforeToday()
	}

	taskDeleted := taskManager.DeleteTask(tasks, index)
	if taskDeleted == nil {
		return errors.New(fmt.Sprintf("Bad index \"%s\"", index))
	}

	if cmdManager.TimeSet {
		taskDeleted.DueDate = cmdManager.DueDate
	} else {
		taskDeleted.DueDate = taskDeleted.DueDate.AddDate(0, 0, 1)
	}

	err := taskManager.SaveTask(taskDeleted)
	return err
}

// -L, forwards the call and sets the prompt skip
func (cmdManager *CommandManager) GetCategories(taskManager *TaskManager) Categories {
	cmdManager.SkipTaskCreationPrompt = true
	return taskManager.GetCategories()
}

// -A
func (cmdManager *CommandManager) GetAuditLog(taskManager *TaskManager) Records {
	cmdManager.SkipTaskCreationPrompt = true

	records := taskManager.AuditRecords()
	if !cmdManager.TimeSet {
		return records
	}

	year, month, day := cmdManager.DueDate.Year(),
		cmdManager.DueDate.Month(),
		cmdManager.DueDate.Day()

	midnight := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	var filteredRecords Records
	for _, record := range records {
		if record.DateCompleted.After(midnight) {
			filteredRecords = append(filteredRecords, record)
		}
	}
	return filteredRecords
}

// -a, at the end if no action taken. Only call at the end, if tasks should be returned
// we will do so.
func (cmdManager *CommandManager) GetTasksIfAll(taskManager *TaskManager) Tasks {
	if cmdManager.Listing == LISTING_ALL && !cmdManager.SkipTaskCreationPrompt {
		cmdManager.SkipTaskCreationPrompt = true
		allTasks := GetTasks(taskManager)
		return *allTasks
	}
	return Tasks{}
}

func (cmdManager *CommandManager) CreateTask(taskManager *TaskManager,
	input string) (*Task, error) {
	task, err := NewTask(input, cmdManager.DueDate,
		cmdManager.Repeat, cmdManager.OverdueDays)
	if err != nil {
		return nil, err
	}

	err = taskManager.SaveTask(&task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}
