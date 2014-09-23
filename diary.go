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
  Content     string      `datastore:",noindex"`
}

type Diary struct {
  CreatedAt   time.Time
  Author      string
  Token       string
}

const REPLY_TO_ADDRESS string = "%s@commanigy-diary.appspotmail.com"

func init() {
  http.HandleFunc("/", root)
  http.HandleFunc("/signup", signup)
  http.HandleFunc("/signout", signout)
  http.HandleFunc("/write", write)
  http.HandleFunc("/diary", diary)
  http.HandleFunc("/mails/daily", dailyMail)
  http.HandleFunc("/_ah/mail/", incomingMail)
}

func root(w http.ResponseWriter, r *http.Request) {
  t, err := template.ParseFiles("templates/index.html")
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  t.Execute(w, nil)
}

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

    g := Diary {
      CreatedAt: time.Now().UTC(),
      Author: u.Email,
      Token: fmt.Sprintf("%x", md5.New().Sum(nil)),
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

func diary(w http.ResponseWriter, r *http.Request) {
  c := appengine.NewContext(r)

  u := user.Current(c)

  ancestorKey := datastore.NewKey(c, "Diary", u.Email, 0, nil)

  var diary Diary
  if err := datastore.Get(c, ancestorKey, &diary); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  q := datastore.NewQuery("DiaryEntry").Ancestor(ancestorKey).Order("-CreatedAt").Limit(5)
  entries := make([]DiaryEntry, 0, 5)
  if _, err := q.GetAll(c, &entries); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  data := struct {
    Diary Diary
    DiaryEntries []DiaryEntry
    EmailAddress string
  } {
    diary,
    entries,
    fmt.Sprintf(REPLY_TO_ADDRESS, diary.Token),
  }

  t, err := template.ParseFiles("templates/diaries/index.html")
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  t.Execute(w, data)  
}

// for testing purposes
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

  http.Redirect(w, r, "/diary", http.StatusFound)
}

func dailyMail(w http.ResponseWriter, r *http.Request) {
  c := appengine.NewContext(r)

  query := datastore.NewQuery("Diary").Order("-CreatedAt")
  for t := query.Run(c); ; {
    var diary Diary
    _, err := t.Next(&diary)
    if err == datastore.Done {
      break
    }
    if err != nil {
      c.Errorf("Failed to populate diary: %v", err)
      continue
    }

    token := diary.Token

    const layout = "Monday, Jan 2"
    today := time.Now().UTC().Format(layout)

    url := "http://diary.commanigy.com/"
    msg := &appmail.Message{
      Sender:  "Diary Support <theill@gmail.com>",
      ReplyTo: fmt.Sprintf(REPLY_TO_ADDRESS, token),
      To:      []string{diary.Author},
      Subject: fmt.Sprintf("It's %s - How did your day go?", today),
      Body:    fmt.Sprintf(dailyMailMessage, url),
    }
    if err := appmail.Send(c, msg); err != nil {
      c.Errorf("Couldn't send email: %v", err)
    }

    c.Infof("Daily mail send to %s", diary.Author)
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
    c.Errorf("Failed to read stream body: %s", err)
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  
  msg, err2 := mail.ReadMessage(bytes.NewReader(b.Bytes()))
  if err2 != nil {
    c.Errorf("Failed to read message: %s", err2)
    http.Error(w, err2.Error(), http.StatusInternalServerError)
    return
  }

  var bodyBuffer bytes.Buffer
  if _, err4 := bodyBuffer.ReadFrom(msg.Body); err4 != nil {
    c.Errorf("Failed to read body: %s", err4)
    http.Error(w, err4.Error(), http.StatusInternalServerError)
    return
  }

  addresses, err := msg.Header.AddressList("To")
  if err != nil {
    c.Errorf("Failed to parse addresses: %s", err)
    return
  }

  var diaryEntryKey *datastore.Key

  for _, address := range addresses {
    token := tokenFromEmailAddress(address.Address)
    c.Infof("Received mail with token %s", token)

    q := datastore.NewQuery("Diary").Filter("Token =", token).KeysOnly()

    diaryKeys, err3 := q.GetAll(c, nil)
    if err3 != nil {
      c.Errorf("Failed to read diary: %s", err3)
      http.Error(w, err3.Error(), http.StatusInternalServerError)
      continue
    }

    if len(diaryKeys) == 0 {
      continue
    }

    diaryEntryKey = datastore.NewIncompleteKey(c, "DiaryEntry", diaryKeys[0])
  }

  if diaryEntryKey == nil {
    c.Errorf("No diary found")
    return
  }

  diaryEntry := DiaryEntry {
    CreatedAt: time.Now().UTC(),
    Content: bodyBuffer.String(),
  }

  _, err5 := datastore.Put(c, diaryEntryKey, &diaryEntry)
  if err5 != nil {
    c.Errorf("Failed to insert diary entry %s", err5)
    http.Error(w, err5.Error(), http.StatusInternalServerError)
    return
  }

  c.Infof("Added new diary entry for key %s", diaryEntryKey)  
}

const dailyMailMessage = `
Just reply to this email with your entry.

(include this one if possible) Remember this? One year ago you wrote...

%s

<a href="/latest">Past entries</a> | <a href="/settings/emailfrequency">Unsubscribe</a>
`
