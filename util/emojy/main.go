package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var (
	token = flag.String("token", "", "Slack API token")
	path  = flag.String("path", "./files", "Path to save files")
)

func main() {
	flag.Parse()

	if *token == "" {
		log.Printf("No token provided.")
		return
	}

	files := getFiles()
	for n, f := range files {
		downloadFile(n, f)
	}
}

func getFiles() map[string]string {
	files := fileResp{}

	log.Printf("Getting files")
	body := mkReq("https://slack.com/api/emoji.list",
		"token", *token,
	)

	err := json.Unmarshal(body, &files)
	checkErr(err)

	log.Printf("Ok: %v", files.Ok)
	if !files.Ok {
		log.Println(files)
	}

	return files.Files
}

func downloadFile(n, f string) {
	url := strings.Replace(f, "\\", "", -1) // because fuck slack

	if strings.HasPrefix(url, "alias:") {
		log.Printf("Skipping alias: %s", url)
		return
	}

	fileNameParts := strings.Split(url, `/`)
	ext := fileNameParts[len(fileNameParts)-1]
	fileNameParts = strings.Split(ext, ".")
	ext = fileNameParts[len(fileNameParts)-1]

	fname := filepath.Join(*path, n+"."+ext)

	log.Printf("Downloading from: %s", url)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	checkErr(err)
	req.Header.Add("Authorization", "Bearer "+*token)

	resp, err := client.Do(req)

	checkErr(err)
	defer resp.Body.Close()
	out, err := os.Create(fname)
	checkErr(err)
	defer out.Close()
	io.Copy(out, resp.Body)

	log.Printf("Downloaded %s", f)
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func mkReq(path string, arg ...string) []byte {
	if len(arg)%2 != 0 {
		log.Fatal("Bad request arg number.")
	}

	u, err := url.Parse(path)
	checkErr(err)

	q := u.Query()
	for i := 0; i < len(arg); i += 2 {
		key := arg[i]
		val := arg[i+1]
		if val != "" {
			q.Set(key, val)
		}
	}
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	checkErr(err)

	body, err := ioutil.ReadAll(resp.Body)
	checkErr(err)

	return body
}

type fileResp struct {
	Ok    bool              `json:"ok"`
	Files map[string]string `json:"emoji"`
}
