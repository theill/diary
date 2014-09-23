package commanigy

import (
  "fmt"
  "html/template"
  "net/http"
  "net/mail"
  "time"
  "bytes"
  "strings"
  "crypto/md5"

  "appengine"
  "appengine/datastore"
  "appengine/user"
  appmail "appengine/mail"
)

type DiaryEntry struct {
  CreatedAt   time.Time
  Content     string
}

type Diary struct {
  CreatedAt   time.Time
  Author      string
  Token       string
  // Entries     []DiaryEntry
}

func init() {
  http.HandleFunc("/", root)
  // http.HandleFunc("/sign", sign)
  http.HandleFunc("/signup", signup)
  // http.HandleFunc("/signin", signin)
  http.HandleFunc("/signout", signout)
  http.HandleFunc("/write", write)
  http.HandleFunc("/diary", diary)
  http.HandleFunc("/mails/daily", dailyMail)
  http.HandleFunc("/_ah/mail/", incomingMail)
}

// // guestbookKey returns the key used for all guestbook entries.
// func guestbookKey(c appengine.Context) *datastore.Key {
//   // The string "default_guestbook" here could be varied to have multiple guestbooks.
//   return datastore.NewKey(c, "Guestbook", "default_guestbook", 0, nil)
// }

func root(w http.ResponseWriter, r *http.Request) {
  if err := guestbookTemplate.Execute(w, nil); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}

var guestbookTemplate = template.Must(template.New("book").Parse(`
<html>
  <head>
    <title>Diary</title>
  </head>
  <body>
    <h1>Diary</h1>
    <p>Welcome to Diary - an open-source OhLife replacement programmed in GO</p>

    <form action="/signup" method="post">
      <div><input type="submit" value="Sign up"></div>
    </form>
  </body>
</html>
`))

func signup(w http.ResponseWriter, r *http.Request) {
  c := appengine.NewContext(r)

  u := user.Current(c)
  if u == nil {
    url, err := user.LoginURL(c, r.URL.String())
    if err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
    }
    w.Header().Set("Location", url)
    w.WriteHeader(http.StatusFound)
    return
  }

  ancestorKey := datastore.NewKey(c, "Diary", u.Email, 0, nil)

  var diary *Diary
  err := datastore.Get(c, ancestorKey, diary)
  if err != nil {
    // new user

    h := md5.New()

    g := Diary {
      CreatedAt: time.Now().UTC(),
      Author: u.Email,
      Token: fmt.Sprintf("%x", h.Sum(nil)),
    }

    key := datastore.NewKey(c, "Diary", u.Email, 0, nil)
    _, err := datastore.Put(c, key, &g)
    if err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
    }
  }

  http.Redirect(w, r, "/diary", http.StatusFound)
}

func signout(w http.ResponseWriter, r *http.Request) {
  c := appengine.NewContext(r)
  u := user.Current(c)
  if u != nil {
    url, err := user.LogoutURL(c, r.URL.String())
    if err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
    }
    w.Header().Set("Location", url)
    w.WriteHeader(http.StatusFound)
    return
  }
  http.Redirect(w, r, "/", http.StatusFound)
}

// func signin(w http.ResponseWriter, r *http.Request) {
//   c := appengine.NewContext(r)
//   g := Greeting {
//     Content: r.FormValue("content"),
//     Date:    time.Now(),
//   }
//   if u := user.Current(c); u != nil {
//     g.Author = u.String()
//   }
//   // We set the same parent key on every Greeting entity to ensure each Greeting
//   // is in the same entity group. Queries across the single entity group
//   // will be consistent. However, the write rate to a single entity group
//   // should be limited to ~1/second.
//   key := datastore.NewIncompleteKey(c, "Greeting", guestbookKey(c))
//   _, err := datastore.Put(c, key, &g)
//   if err != nil {
//     http.Error(w, err.Error(), http.StatusInternalServerError)
//     return
//   }
//   http.Redirect(w, r, "/", http.StatusFound)
// }

// func sign(w http.ResponseWriter, r *http.Request) {
//   c := appengine.NewContext(r)
//   g := Greeting{
//     Content: r.FormValue("content"),
//     Date:    time.Now(),
//   }
//   if u := user.Current(c); u != nil {
//     g.Author = u.String()
//   }
//   // We set the same parent key on every Greeting entity to ensure each Greeting
//   // is in the same entity group. Queries across the single entity group
//   // will be consistent. However, the write rate to a single entity group
//   // should be limited to ~1/second.
//   key := datastore.NewIncompleteKey(c, "Greeting", guestbookKey(c))
//   _, err := datastore.Put(c, key, &g)
//   if err != nil {
//     http.Error(w, err.Error(), http.StatusInternalServerError)
//     return
//   }
//   http.Redirect(w, r, "/", http.StatusFound)
// }

func diary(w http.ResponseWriter, r *http.Request) {
  c := appengine.NewContext(r)

  u := user.Current(c)

  ancestorKey := datastore.NewKey(c, "Diary", u.Email, 0, nil)

  q := datastore.NewQuery("DiaryEntry").Ancestor(ancestorKey).Order("-CreatedAt").Limit(10)
  greetings := make([]DiaryEntry, 0, 10)
  if _, err := q.GetAll(c, &greetings); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  if err := diaryTemplate.Execute(w, greetings); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}

var diaryTemplate = template.Must(template.New("book").Parse(`
<html>
  <head>
    <title>Diary</title>
  </head>
  <body>
    {{range .}}
      <h3>{{ .CreatedAt }}</h3>
      <pre>{{ .Content }}</pre>
    {{end}}
    <hr />
    <form action="/signout" method="post">
      <div><input type="submit" value="Sign out"></div>
    </form>
  </body>
</html>
`))

// func findDiaryByToken(c appengine.Context, token string) (*Diary, error) {
//   ancestorKey := datastore.NewKey(c, "Diary", "default_diary", 0, nil)
//   q := datastore.NewQuery("Diary").Ancestor(ancestorKey).Filter("Token =", token).Limit(1)

//   var diaryKeys []*datastore.Key

//   diaries := make([]Diary, 0, 1)
//   diaryKeys, err := q.GetAll(c, &diaries)
//   if err != nil {
//     return nil, err
//   }

//   return &diaries[0], nil
// }

func write(w http.ResponseWriter, r *http.Request) {
  c := appengine.NewContext(r)

  u := user.Current(c)

  ancestorKey := datastore.NewKey(c, "Diary", u.Email, 0, nil)

  var diary Diary
  err := datastore.Get(c, ancestorKey, &diary)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  diaryEntry := DiaryEntry {
    CreatedAt: time.Now().UTC(),
    Content: "Whatever",
  }

  key := datastore.NewIncompleteKey(c, "DiaryEntry", ancestorKey)
  _, err3 := datastore.Put(c, key, &diaryEntry)
  if err3 != nil {
    http.Error(w, err3.Error(), http.StatusInternalServerError)
    return
  }


  // ancestorKey := datastore.NewKey(c, "Diary", "default_diary", 0, nil)
  // q := datastore.NewQuery("Diary").Ancestor(ancestorKey).Filter("Author =", u.Email).Limit(1)

  // var diaryKeys []*datastore.Key

  // diaries := make([]Diary, 0, 1)
  // diaryKeys, err := q.GetAll(c, &diaries)
  // if err != nil {
  //   http.Error(w, err.Error(), http.StatusInternalServerError)
  //   return
  // }

  // diaries[0].Entries = append(diaries[0].Entries, DiaryEntry {
  //   CreatedAt: time.Now().UTC(),
  //   Content: "AnotherOne",
  // })
  
  // _, err2 := datastore.Put(c, diaryKeys[0], &diaries[0])
  // if err2 != nil {
  //   http.Error(w, err2.Error(), http.StatusInternalServerError)
  //   return
  // }

  http.Redirect(w, r, "/diary", http.StatusFound)
}

func dailyMail(w http.ResponseWriter, r *http.Request) {
  c := appengine.NewContext(r)

  q := datastore.NewQuery("Diary").Order("-CreatedAt")
  
  for t := q.Run(c); ; {
    var x Diary
    _, err := t.Next(&x)
    if err == datastore.Done {
      break
    }
    if err != nil {
      // serveError(c, w, err)
      return
    }

    token := x.Token

    const layout = "Monday, Jan 2"
    t := time.Now().UTC()
    today := t.Format(layout)

    url := "http://diary.commanigy.com/"
    msg := &appmail.Message{
      Sender:  "Diary Support <theill@gmail.com>",
      ReplyTo: fmt.Sprintf("%s@commanigy-diary.appspotmail.com", token),
      To:      []string{x.Author},
      Subject: fmt.Sprintf("It's %s - How did your day go?", today),
      Body:    fmt.Sprintf(dailyMailMessage, url),
    }
    if err := appmail.Send(c, msg); err != nil {
      c.Errorf("Couldn't send email: %v", err)
    }

    c.Infof("Daily mail send to %s", x.Author)
  }
}

func tokenFromEmailAddress(emailAddress string) (string) {
  return strings.Split(emailAddress, "@")[0]
}

func incomingMail(w http.ResponseWriter, r *http.Request) {
  // TODO: parse mail body and extract important part
  // TODO: discard messages which are too big

  c := appengine.NewContext(r)
  defer r.Body.Close()
  var b bytes.Buffer
  if _, err := b.ReadFrom(r.Body); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  
  msg, err2 := mail.ReadMessage(bytes.NewReader(b.Bytes()))
  if err2 != nil {
    http.Error(w, err2.Error(), http.StatusInternalServerError)
    return
  }

  var bodyBuffer bytes.Buffer
  if _, err4 := bodyBuffer.ReadFrom(msg.Body); err4 != nil {
    http.Error(w, err4.Error(), http.StatusInternalServerError)
    return
  }

  token := tokenFromEmailAddress(msg.Header.Get("To"))
  c.Infof("Received mail with token %s", token)  

  q := datastore.NewQuery("Diary").Filter("Token =", token).KeysOnly()

  diaryKeys, err3 := q.GetAll(c, nil)
  if err3 != nil {
    http.Error(w, err3.Error(), http.StatusInternalServerError)
    return
  }

  diaryEntry := DiaryEntry {
    CreatedAt: time.Now().UTC(),
    Content: bodyBuffer.String(),
  }

  diaryEntryKey := datastore.NewIncompleteKey(c, "DiaryEntry", diaryKeys[0])
  _, err5 := datastore.Put(c, diaryEntryKey, &diaryEntry)
  if err5 != nil {
    http.Error(w, err5.Error(), http.StatusInternalServerError)
    return
  }

  c.Infof("Added new diary entry for key %s", diaryEntryKey)  
}

const dailyMailMessage = `
Just reply to this email with your entry.

(include this one if possible) Remember this? One year ago you wrote...

%s

Past entries | Unsubscribe
`
