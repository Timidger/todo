package todo

import (
	"errors"
	"fmt"
	"os/exec"
	"path"
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
func (cmd_manager *CommandManager) SetDueDate(new_due_date time.Time) {
	cmd_manager.DueDate = new_due_date
	cmd_manager.TimeSet = true
}

func humany_time(day string) (time.Time, error) {
	today := time.Now()
	relative_day := 0
	cur_weekday := int(today.Weekday())
	switch strings.Title(day) {
	case "Sunday":
		relative_day = 0
	case "Monday":
		relative_day = 1
	case "Tuesday":
		relative_day = 2
	case "Wednesday":
		relative_day = 3
	case "Thursday":
		relative_day = 4
	case "Friday":
		relative_day = 5
	case "Saturday":
		relative_day = 6
	case "Tomorrow":
		relative_day = (cur_weekday + 1) % 7
	case "Today":
		return today, nil
	case "Yesterday":
		return today.AddDate(0, 0, -1), nil
	default:
		return today, errors.New(fmt.Sprintf("Invalid human-y day %s", day))
	}
	if cur_weekday < relative_day {
		return today.AddDate(0, 0, int(relative_day-cur_weekday)), nil
	} else {
		return today.AddDate(0, 0, 7-(cur_weekday-relative_day)), nil
	}
}

// -t
func (cmd_manager *CommandManager) SetDueDateRelative(new_due_date string) error {
	cmd_manager.TimeSet = true
	human_due_date, err := humany_time(new_due_date)
	if err != nil {
		cmd_manager.DueDate = human_due_date
		return nil
	}
	// Attempt to use date as a fallback, using ISO-8601 format
	var due_date time.Time
	out, err := exec.Command("/usr/bin/date", "--iso-8601", "-d", new_due_date).Output()
	if err != nil {
		return errors.New(fmt.Sprintf("/usr/bin/date failed: %v", err))
	}
	new_due_date = strings.ReplaceAll(strings.TrimSpace(string(out)), "-", "/")
	due_date, err = time.Parse(EXPLICIT_TIME_FORMAT, new_due_date+" EDT")
	if err != nil {
		return errors.New(fmt.Sprintf("Bad date: %s", new_due_date))
	}
	cmd_manager.DueDate = due_date
	return nil
}

// -l
func (cmd_manager *CommandManager) GetTasks(task_manager *TaskManager) (Tasks, error) {
	all_tasks := GetTasks(task_manager)
	cmd_manager.Listing = LISTING_DAY
	cmd_manager.SkipTaskCreationPrompt = true

	var tasks Tasks
	if cmd_manager.DueDate.Before(time.Now()) && !cmd_manager.TimeSet {
		tasks = all_tasks.FilterTasksDueBeforeToday()
		if len(tasks) == 0 {
			tasks = *all_tasks
		}
	} else {
		tasks = all_tasks.FilterTasksDueOnDay(cmd_manager.DueDate)
	}

	return tasks, nil
}

// What -a should be, don't list until we know we aren't gonna need to pipe
func (cmd_manager *CommandManager) UseAllTasks() {
	cmd_manager.Listing = LISTING_ALL
}

// -s
func (cmd_manager *CommandManager) SkipTask(task_manager *TaskManager, index string) error {
	skip_task := GetTasks(task_manager).GetByHash(index)
	if skip_task.Repeat == nil {
		return errors.New("Can only skip repeat tasks")
	}
	_, err := cmd_manager.delete_task_helper(task_manager, index, false, true)
	return err
}

// -D (true) and -d (false)
func (cmd_manager *CommandManager) DeleteTask(task_manager *TaskManager, index string,
	force_delete bool) (*Task, error) {

	task, err := cmd_manager.delete_task_helper(task_manager, index, force_delete, false)
	if err == nil && task == nil {
		panic("At least one value was expected to be non-nil")
	}
	return task, err
}

// helper for -D, -d, and -s
func (cmd_manager *CommandManager) delete_task_helper(task_manager *TaskManager, index string,
	force_delete, skip_repeat bool) (*Task, error) {

	if force_delete && skip_repeat {
		panic("force_delete and skip_repeat cannot both be true")
	}

	all_tasks := GetTasks(task_manager)
	cmd_manager.SkipTaskCreationPrompt = true
	var task_deleted *Task

	switch cmd_manager.Listing {
	case LISTING_DAY:
		var tasks Tasks
		if cmd_manager.DueDate.Before(time.Now()) {
			// NOTE This is a special case: we want everything due today
			// or before today with this call..
			tasks = all_tasks.FilterTasksDueBeforeToday()
		} else {
			tasks = all_tasks.FilterTasksDueOnDay(cmd_manager.DueDate)
		}
		if len(tasks) != 0 {
			task_deleted = task_manager.DeleteTask(tasks, index)
			if task_deleted != nil && task_deleted.Repeat == nil {
				all_tasks.RemoveFirst(*task_deleted)
			}
			break
		}
		// If there are no tasks today then we must try to delete based
		// on all tasks. This lets you use it like -l when there are no
		// tasks today.
		fallthrough
	case LISTING_ALL:
		task_deleted = (task_manager).DeleteTask(*all_tasks, index)
		if task_deleted != nil {
			all_tasks.RemoveFirst(*task_deleted)
		}
	}

	if task_deleted == nil {
		return nil, errors.New(fmt.Sprintf("Bad index \"%s\"", index))
	}

	if !force_delete && task_deleted.Repeat != nil {
		// Recreate the task if it has a repeat.

		delay, err := time.ParseDuration(*task_deleted.Repeat)
		if err == nil {
			task_deleted.Due_date = task_deleted.Due_date.Add(delay)
		} else {
			// Must be human-y
			days := strings.Split(*task_deleted.Repeat, ",")
			if len(days) == 0 {
				return nil, errors.New("Empty repeat is not allowed")
			}
			if len(days) == 1 {
				delay, err := humany_time(days[0])
				if err != nil {
					return nil, err
				}
				task_deleted.Due_date = delay
			} else {
				var cur_delay time.Time
				var err error
				delay_set := false
				for _, day := range days {
					cur_delay, err = humany_time(day)
					if err != nil {
						continue
					}
					if cur_delay.After(time.Now()) {
						task_deleted.Due_date = cur_delay
						err = nil
						delay_set = true
						break
					}
				}
				if !delay_set {
					// Get the earliest next date -- which is guaranteed to be the first one
					delay, err := humany_time(days[0])
					if err != nil {
						return nil, err
					}
					task_deleted.Due_date = delay
				}
				if err != nil {
					return nil, err
				}
			}

		}
		if err := task_manager.SaveTask(task_deleted); err != nil {
			return nil, err
		}
	}

	// Log in the audit log
	if !force_delete && !skip_repeat {
		original_StorageDirectory := task_manager.StorageDirectory
		if task_deleted.category != nil {
			task_manager.StorageDirectory = path.Join(task_manager.StorageDirectory, *task_deleted.category)
		}
		task_manager.AuditLog(*task_deleted, cmd_manager.DueDate, cmd_manager.Annotation)
		task_manager.StorageDirectory = original_StorageDirectory
	}

	return task_deleted, nil
}

// -r
func (cmd_manager *CommandManager) SetRepeatHumany(days string) error {
	for _, day := range strings.Split(days, ",") {
		switch strings.Title(day) {
		case "Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday":
			continue
		default:
			return errors.New(fmt.Sprintf("Invalid repeat day %s", day))
		}
	}
	cmd_manager.Repeat = &days
	return nil
}

// -r
func (cmd_manager *CommandManager) SetRepeat(days int) error {
	if days <= 0 {
		return errors.New("Repeat time must be a positive, non-zero number")
	}

	hours, err := time.ParseDuration(fmt.Sprintf("%dh", days*24))
	if err != nil {
		return err
	}
	hours_str := string(hours)
	cmd_manager.Repeat = &hours_str

	return nil
}

// -n
func (cmd_manager *CommandManager) SetDelay(days int) error {
	if days <= 0 {
		return errors.New("Delay time must be a positive, non-zero number")
	}
	cmd_manager.OverdueDays = int(days)
	return nil
}

// -x
func (cmd_manager *CommandManager) DelayTask(task_manager *TaskManager, index string) error {
	cmd_manager.SkipTaskCreationPrompt = true
	// Always do a re-read for delays. Makes them both more expensive and multiple delays work.
	tasks = nil
	all_tasks := GetTasks(task_manager)

	var tasks Tasks
	switch cmd_manager.Listing {
	case LISTING_ALL:
		tasks = *all_tasks
	case LISTING_DAY:
		tasks = all_tasks.FilterTasksDueBeforeToday()
	}

	task_deleted := task_manager.DeleteTask(tasks, index)
	if task_deleted == nil {
		return errors.New(fmt.Sprintf("Bad index \"%s\"", index))
	}

	if cmd_manager.TimeSet {
		task_deleted.Due_date = cmd_manager.DueDate
	} else {
		task_deleted.Due_date = task_deleted.Due_date.AddDate(0, 0, 1)
	}

	err := task_manager.SaveTask(task_deleted)
	return err
}

// -L, forwards the call and sets the prompt skip
func (cmd_manager *CommandManager) GetCategories(task_manager *TaskManager) Categories {
	cmd_manager.SkipTaskCreationPrompt = true
	return task_manager.GetCategories()
}

// -A
func (cmd_manager *CommandManager) GetAuditLog(task_manager *TaskManager) Records {
	cmd_manager.SkipTaskCreationPrompt = true

	records := task_manager.AuditRecords()
	if !cmd_manager.TimeSet {
		return records
	}

	year, month, day := cmd_manager.DueDate.Year(),
		cmd_manager.DueDate.Month(),
		cmd_manager.DueDate.Day()

	midnight := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	var filtered_records Records
	for _, record := range records {
		if record.DateCompleted.After(midnight) {
			filtered_records = append(filtered_records, record)
		}
	}
	return filtered_records
}

// -a, at the end if no action taken. Only call at the end, if tasks should be returned
// we will do so.
func (cmd_manager *CommandManager) GetTasksIfAll(task_manager *TaskManager) Tasks {
	if cmd_manager.Listing == LISTING_ALL && !cmd_manager.SkipTaskCreationPrompt {
		cmd_manager.SkipTaskCreationPrompt = true
		all_tasks := GetTasks(task_manager)
		return *all_tasks
	}
	return Tasks{}
}

func (cmd_manager *CommandManager) CreateTask(task_manager *TaskManager,
	input string) (*Task, error) {
	task, err := NewTask(input, cmd_manager.DueDate,
		cmd_manager.Repeat, cmd_manager.OverdueDays)
	if err != nil {
		return nil, err
	}

	err = task_manager.SaveTask(&task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}
