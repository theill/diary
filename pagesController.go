package ohdiary

import (
  // "fmt"
  "net/http"

  "github.com/martini-contrib/render"

  "appengine"
  // "appengine/datastore"
  "appengine/user"
)

func PagesControllerIndex(r render.Render, req *http.Request) {
  c := appengine.NewContext(req)

  u := user.Current(c)
  
  data := struct {
    CurrentUser string
  } {
    "",
  }

  if u != nil {
    data.CurrentUser = u.String()
  }

  r.HTML(200, "pages/index", data)
}
