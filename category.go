package main

import "fmt"

type Category struct {
	Name  string
	Tasks int
}

func (category Category) String() string {
	return fmt.Sprintf("%-20s %d tasks", category.Name, category.Tasks)
}

type Categories []Category

func (categories Categories) Len() int {
	return len(categories)
}

func (categories Categories) Less(i, j int) bool {
	return categories[i].Tasks < categories[j].Tasks
}

func (categories Categories) Swap(i, j int) {
	categories[i], categories[j] = categories[j], categories[i]
}
