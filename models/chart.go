package models

import (
	"fmt"
	"time"

	"github.com/jinzhu/now"
)

type Chart struct {
	Total string
	Date  time.Time
}

type UserChart struct {
	Total  string
	Seller string
}

/*
date format 2015-01-23
*/
func GetChartData(table, start, end, currentUser string) (res []Chart) {
	startdate, err := now.Parse(start)
	if err != nil {
		return
	}

	enddate, err := now.Parse(end)
	if err != nil || enddate.UnixNano() < startdate.UnixNano() {
		enddate = now.EndOfDay()
	} else {
		enddate = enddate.AddDate(0, 0, 1)
	}

	if currentUser != "" {
		DB.Table(table).
			Where("deleted_at IS NULL AND created_at > ? AND created_at < ? AND seller = ?", startdate, enddate, currentUser).
			Select("date(created_at) as date, count(*) as total").
			Group("date(created_at)").
			Order("date(created_at)").
			Scan(&res)
	} else {
		DB.Table(table).
			Where("deleted_at IS NULL AND created_at > ? AND created_at < ?", startdate, enddate).
			Select("date(created_at) as date, count(*) as total").
			Group("date(created_at)").
			Order("date(created_at)").
			Scan(&res)
	}

	return
}

func GetChartDataOfSum(table, field, start, end, currentUser string) (res []Chart) {
	startdate, err := now.Parse(start)
	if err != nil {
		return
	}

	enddate, err := now.Parse(end)
	if err != nil || enddate.UnixNano() < startdate.UnixNano() {
		enddate = now.EndOfDay()
	} else {
		enddate = enddate.AddDate(0, 0, 1)
	}

	if currentUser != "" {
		DB.Table(table).
			Where("deleted_at IS NULL AND created_at > ? AND created_at < ? AND seller = ?", startdate, enddate, currentUser).
			Select(fmt.Sprintf("date(created_at) as date, sum(%s) as total", field)).
			Group("date(created_at)").
			Order("date(created_at)").
			Scan(&res)
	} else {
		DB.Table(table).
			Where("deleted_at IS NULL AND created_at > ? AND created_at < ?", startdate, enddate).
			Select(fmt.Sprintf("date(created_at) as date, sum(%s) as total", field)).
			Group("date(created_at)").
			Order("date(created_at)").
			Scan(&res)
	}

	return
}

func GetUserChartDataOfSum(table, field, start, end string) (res []UserChart) {
	startdate, err := now.Parse(start)
	if err != nil {
		return
	}
	
	enddate, err := now.Parse(end)
	if err != nil || enddate.UnixNano() < startdate.UnixNano() {
		enddate = now.EndOfDay()
	} else {
		enddate = enddate.AddDate(0, 0, 1)
	}

	DB.Table(table).
		Where("deleted_at IS NULL AND created_at > ? AND created_at < ?", startdate, enddate).
		Select(fmt.Sprintf("seller, sum(%s) as total", field)).
		Group("seller").
		Scan(&res)
	
	return
}
