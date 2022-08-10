package application

import "time"

type Application struct {
	AppID          string
	BundleID       string
	Name           string
	Version        string
	Author         string
	Icon           string
	CreateDateTime time.Time
	UpdateDateTime time.Time
}
