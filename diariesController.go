package ohdiary

import (
  "fmt"
  "net/http"

  "github.com/martini-contrib/render"

  "appengine"
  "appengine/datastore"
  "appengine/user"
)

func DiariesControllerIndex(r render.Render, req *http.Request) {
    c := appengine.NewContext(req)

    u := user.Current(c)

    ancestorKey := datastore.NewKey(c, "Diary", u.Email, 0, nil)

    var diary Diary
    if err := datastore.Get(c, ancestorKey, &diary); err != nil {
      return
    }

    q, _ := datastore.NewQuery("DiaryEntry").Ancestor(ancestorKey).Count(c)

    data := struct {
      Diary Diary
      DiaryEntriesCount int
      EmailAddress string
      CurrentUser string
    } {
      diary,
      q,
      fmt.Sprintf(REPLY_TO_ADDRESS, diary.Token),
      u.String(),
    }

    r.HTML(200, "diaries/index", data)
  }