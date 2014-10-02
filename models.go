package ohdiary

import (
  "time"
)

type AppConfiguration struct {
  SendGridUser    string
  SendGridKey     string
}

type DiaryEntry struct {
  CreatedAt       time.Time
  Content         string      `datastore:",noindex"`
}

type Diary struct {
  CreatedAt       time.Time
  Author          string
  Token           string
  TimeZone        string      // "Europe/Berlin"
  TimeOffset      int         // 10
}
