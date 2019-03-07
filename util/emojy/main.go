package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	token = flag.String("token", "", "Slack API token")
	path  = flag.String("path", "./files", "Path to save files")
)

func main() {
	flag.Parse()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if *token == "" {
		log.Fatal().Msg("No token provided.")
		return
	}

	files := getFiles()
	for n, f := range files {
		downloadFile(n, f)
	}
}

func getFiles() map[string]string {
	files := fileResp{}

	log.Debug().Msgf("Getting files")
	body := mkReq("https://slack.com/api/emoji.list",
		"token", *token,
	)

	err := json.Unmarshal(body, &files)
	checkErr(err)

	log.Debug().Msgf("Ok: %v", files.Ok)
	if !files.Ok {
		log.Debug().Msgf("%+v", files)
	}

	return files.Files
}

func downloadFile(n, f string) {
	url := strings.Replace(f, "\\", "", -1) // because fuck slack

	if strings.HasPrefix(url, "alias:") {
		log.Debug().Msgf("Skipping alias: %s", url)
		return
	}

	fileNameParts := strings.Split(url, `/`)
	ext := fileNameParts[len(fileNameParts)-1]
	fileNameParts = strings.Split(ext, ".")
	ext = fileNameParts[len(fileNameParts)-1]

	fname := filepath.Join(*path, n+"."+ext)

	log.Debug().Msgf("Downloading from: %s", url)

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

	log.Debug().Msgf("Downloaded %s", f)
}

func checkErr(err error) {
	if err != nil {
		log.Fatal().Err(err)
	}
}

func mkReq(path string, arg ...string) []byte {
	if len(arg)%2 != 0 {
		log.Fatal().Msg("Bad request arg number.")
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
