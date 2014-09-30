package ohdiary

import (
  "net/http"
  "net/mail"
  "time"
  "bytes"
  "strings"
  "encoding/base64"

  "io"
  "io/ioutil"
  "mime"
  "mime/multipart"
  
  "appengine"
  "appengine/datastore"
)

func incomingMail(w http.ResponseWriter, r *http.Request) {
  // TODO: parse mail body and extract important part
  // TODO: discard messages which are too big
  // TODO: include "remember, one (week|month|year) ago you wrote" in mails

  c := appengine.NewContext(r)

  defer r.Body.Close()
  var b bytes.Buffer
  if _, err := b.ReadFrom(r.Body); err != nil {
    c.Errorf("Failed to read stream body: %s", err)
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  
  msg, err2 := mail.ReadMessage(strings.NewReader(b.String()))
  if err2 != nil {
    c.Errorf("Failed to read message: %s", err2)
    http.Error(w, err2.Error(), http.StatusInternalServerError)
    return
  }

  // var bodyBuffer bytes.Buffer
  // if _, err4 := bodyBuffer.ReadFrom(msg.Body); err4 != nil {
  //   c.Errorf("Failed to read body: %s", err4)
  //   http.Error(w, err4.Error(), http.StatusInternalServerError)
  //   return
  // }

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

  // msg = &mail.Message{
  //   Header: map[string][]string{
  //     "Content-Type": []string{"multipart/mixed; boundary=foo"},
  //   },
  //   Body: strings.NewReader(bodyBuffer.String()),
  // }

  content, err := parseMailBody(c, msg)
  if err != nil {
    c.Errorf("Failed to parse mail body: %s", err)
    return
  }

  if len(content) == 0 {
    c.Infof("No content found in posted mail")
    return
  }

  diaryEntry := DiaryEntry {
    CreatedAt: time.Now().UTC(),
    Content: content,
    // Content: bodyBuffer.String(),
  }

  _, err5 := datastore.Put(c, diaryEntryKey, &diaryEntry)
  if err5 != nil {
    c.Errorf("Failed to insert diary entry %s", err5)
    http.Error(w, err5.Error(), http.StatusInternalServerError)
    return
  }

  c.Infof("Added new diary entry for key %s", diaryEntryKey)  
}

func parseMailBody(c appengine.Context, msg *mail.Message) (string, error) {
  mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
  if err != nil {
    c.Errorf("Failed to parse content type %s", err)
    return "", err
  }

  if strings.HasPrefix(mediaType, "multipart/") {
    mr := multipart.NewReader(msg.Body, params["boundary"])
    for {
      p, err := mr.NextPart()
      if err == io.EOF {
        return "", nil
      }
      if err != nil {
        c.Errorf("Failed to read next part of message: %v", p)
        return "", err
      }
      
      slurp, err := ioutil.ReadAll(p)
      if err != nil {
        c.Errorf("Reading all for slurp %s", err)
        return "", err
      }

      transferEncoding := p.Header.Get("Content-Transfer-Encoding")
      if (transferEncoding == "base64") {
        decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(slurp))

        decodedSlurp, err := ioutil.ReadAll(decoder)
        if err != nil {
          c.Errorf("Failed to decode block")
          return "", err
        }

        return string(decodedSlurp), nil
      }

      c.Infof("Part %q: %q\n", p.Header.Get("Content-Type"), slurp)
      return string(slurp), nil
    }
  }

  return "", nil
}