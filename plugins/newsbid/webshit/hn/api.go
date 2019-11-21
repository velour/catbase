package hn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
)

const BASE = "https://hacker-news.firebaseio.com/v0/"

func get(url string) (*http.Response, error) {
	c := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "catbase/1.0")
	return c.Do(req)
}

func GetItem(id int) (Item, error) {
	u := path.Join(BASE, "item", fmt.Sprintf("%d.json", id))
	resp, err := get(u)
	if err != nil {
		return Item{}, err
	}
	dec := json.NewDecoder(resp.Body)
	i := Item{}
	if err := dec.Decode(&i); err != nil {
		return Item{}, err
	}
	return i, nil
}

type Items []Item

func (is Items) Titles() string {
	out := ""
	for i, v := range is {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("<%s|%s>", v.URL, v.Title)
	}
	return out
}
