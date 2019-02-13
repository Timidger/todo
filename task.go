package main

import (
	"time"
)

type Task struct {
	// The "body" content of the task, in Markdown
	body_content string
	// When this task is due and must be done.
	// What this actually means is still up for debate.
	// However, I feel strongly that this should "explode" in some way.
	due_date time.Time
}
