package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

type user struct {
	id   int
	name string
	pass string
}

const (
	authUser = "admin"
	authPass = "password"

	baseURL  = "http://127.0.0.1:8080/"
	roomURL  = baseURL + "room/"
	roomsURL = baseURL + "rooms/"
	userURL  = baseURL + "user/"
	usersURL = baseURL + "users/"

	roomInterval = 15
	roomRounds   = 60

	userTotal = 400
)

var (
	roomID    int
	userCount int64

	limiter *rate.Limiter = rate.NewLimiter(80, 5)
)

func mustPostJSON(url string, body interface{}, auth bool) map[string]interface{} {
	ctx, cancel := context.WithTimeout(context.Background(), roomInterval*time.Second)
	defer cancel()
	if err := limiter.Wait(ctx); err != nil {
		panic(err)
	}

	str, err := json.Marshal(body)
	if err != nil {
		panic("json.Marshal fail")
	}

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(str)))
	req.Header.Set("Content-Type", "application/json")
	if auth {
		req.SetBasicAuth(authUser, authPass)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		panic("http.Post fail")
	}

	jsonMap := make(map[string]interface{})
	jsonStr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic("ioutil.ReadAll fail")
	}

	err = json.Unmarshal([]byte(jsonStr), &jsonMap)
	if err != nil {
		return map[string]interface{}{}
	}

	return jsonMap
}

func roomCreate() {
	body := map[string]interface{}{
		"Interval":   roomInterval,
		"RoundTotal": roomRounds,
	}
	resp := mustPostJSON(roomURL, body, true)

	roomID = int(resp["ID"].(float64))
}

func userRegister() user {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	id := int(atomic.AddInt64(&userCount, 1))
	u := user{
		name: fmt.Sprintf("user-%v", id),
		pass: fmt.Sprintf("password-%v", id),
	}

	body := map[string]interface{}{
		"Username": u.name,
		"Password": u.pass,
	}
	resp := mustPostJSON(usersURL+fmt.Sprintf("%v", roomID), body, false)
	u.id = int(resp["ID"].(float64))

	return u
}

func (u *user) submit() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()

	submit1 := rand.Float64() * 100
	submit2 := rand.Float64() * 100
	body := map[string]interface{}{
		"Password": u.pass,
		"Submit1":  submit1,
		"Submit2":  submit2,
	}

	mustPostJSON(userURL+fmt.Sprintf("%v", u.id), body, false)
	fmt.Println(u.id, submit1, submit2)
}

func main() {
	roomCreate()

	users := make(map[int]*user)
	usersLock := sync.Mutex{}
	var usersWg sync.WaitGroup
	for i := 0; i < userTotal; i++ {
		usersWg.Add(1)
		go func() {
			defer usersWg.Done()
			u := userRegister()
			if u.id == 0 {
				return
			}
			usersLock.Lock()
			defer usersLock.Unlock()
			users[u.id] = &u
			fmt.Println(u)
		}()
	}
	usersWg.Wait()

	for i := 0; i < roomRounds; i++ {
		next := time.Now().Add(roomInterval * time.Second)
		for _, u := range users {
			go u.submit()
		}
		time.Sleep(time.Until(next))
	}
}
