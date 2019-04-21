package main

import (
	"os"
	"testing"
	"time"
)

const TEST_DIRECTORY = "test"

func create_test_manager(t *testing.T) TaskManager {
	var manager TaskManager
	manager.storage_directory = TEST_DIRECTORY
	if len(manager.GetTasks()) > 0 {
		t.Errorf("Test todo folder clashed with existing storage\n")
	}
	return manager
}

func destroy_test_manager(t *testing.T) {
	if err := os.RemoveAll("test"); err != nil {
		t.Errorf("Could not remove test directory: %s\n", err)
	}
}

func TestBasic_todo_storage(t *testing.T) {
	manager := create_test_manager(t)
	defer destroy_test_manager(t)
	manager.AddTask("Test todo item", time.Now())
	task := manager.GetTasks()[0]
	if task.body_content != "Test todo item" {
		t.Errorf("Incorrect todo item message contents: %s\n", task.body_content)
	}
	// todo only stores day information, so it will be before "now" unless it's 12:00
	if !task.due_date.Before(time.Now()) {
		t.Errorf("Bad time for todo task: %s", task.due_date)
	}
}
