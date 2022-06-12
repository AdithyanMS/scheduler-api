package models

type Logic struct {
	ID       int    `json:"id"`
	COUNT    int    `json:"count"`
	STARTIME string `json:"startime"`
	ENDTIME  string `json:"endtime"`
	DURATION int    `json:"duration"`
	INTERVAL int    `json:"interval"`
}
