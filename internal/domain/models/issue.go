package models

type Issue struct {
	ID          int
	Number      int
	Title       string
	Description string
	State       string
	Labels      []string
	Author      string
	URL         string
}
