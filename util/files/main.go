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
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	token     = flag.String("token", "", "Slack API token")
	channel   = flag.String("channel", "", "Slack channel ID")
	limit     = flag.Int("limit", 10000, "Number of items to return")
	types     = flag.String("types", "images,pdfs,video", "Type of object")
	path      = flag.String("path", "./", "Path to save files")
	to        = flag.String("to", "", "Time limit in '2006-01-02T15:04:05Z07:00' format. Default -30d")
	rateLimit = flag.Int("rate", 1, "rate limit in seconds")
)

func main() {
	flag.Parse()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	ticker := time.NewTicker(time.Second * time.Duration(*rateLimit))
	defer ticker.Stop()

	page := 1
	totalFiles := 0

	for {
		files, count := getFiles(page)
		totalFiles += len(files)
		log.Debug().Msgf("Page %d: got %d files, count is %d",
			page, len(files), count)
		for _, f := range files {
			downloadFile(f)
			deleteFile(f)
		}
		if count < 1 {
			break
		}
		<-ticker.C
		if page >= count {
			if totalFiles == 0 {
				break
			}
			totalFiles = 0
			page = 0
		}
		page++
	}
}

func getFiles(page int) ([]slackFile, int) {
	files := fileResp{}

	var toTime time.Time
	var err error

	if *to == "" {
		toTime = time.Now().Add(time.Hour * 24 * 30 * -1)
	} else {
		toTime, err = time.Parse(time.RFC3339, *to)
		if err != nil {
			log.Fatal().Err(err).Msg("Error reading time format")
		}
	}

	log.Debug().Msg("Getting files")
	body := mkReq("https://slack.com/api/files.list",
		"token", *token,
		"count", strconv.Itoa(*limit),
		"types", *types,
		"ts_to", strconv.FormatInt(toTime.Unix(), 10),
		"ts_from", "0",
		"page", strconv.Itoa(page),
	)

	err = json.Unmarshal(body, &files)
	checkErr(err)

	log.Info().
		Int("page", page).
		Int("count", files.Paging.Count).
		Int("total", files.Paging.Total).
		Bool("ok", files.Ok).
		Msg("file result")
	if !files.Ok {
		log.Error().
			Interface("files", files).
			Str("body", string(body)).
			Msg("Error getting files")
		os.Exit(1)
	}

	return files.Files, files.Paging.Pages
}

func deleteFile(f slackFile) {
	body := mkReq("https://slack.com/api/files.delete",
		"token", *token,
		"file", f.ID,
	)

	del := delResp{}
	err := json.Unmarshal(body, &del)

	checkErr(err)
	if !del.Ok {
		log.Fatal().
			Bytes("body", body).
			Str("id", f.ID).
			Msg("Couldn't delete")
	}

	log.Info().
		Str("id", f.ID).
		Msg("Deleted")
}

func downloadFile(f slackFile) {
	url := strings.Replace(f.URLPrivateDownload, "\\", "", -1) // because fuck slack
	fname := filepath.Join(*path, f.ID+f.Name)

	log.Info().
		Str("url", url).
		Msg("Downloading")

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

	log.Info().
		Str("id", f.ID).
		Msg("Downloaded")
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

type delResp struct {
	Ok bool
}

type fileResp struct {
	Ok     bool        `json:"ok"`
	Files  []slackFile `json:"files"`
	Paging struct {
		Count int `json:"count"`
		Total int `json:"total"`
		Page  int `json:"page"`
		Pages int `json:"pages"`
	} `json:"paging"`
}

type slackFile struct {
	ID                 string   `json:"id"`
	Created            int      `json:"created"`
	Timestamp          int      `json:"timestamp"`
	Name               string   `json:"name"`
	Title              string   `json:"title"`
	Mimetype           string   `json:"mimetype"`
	Filetype           string   `json:"filetype"`
	PrettyType         string   `json:"pretty_type"`
	User               string   `json:"user"`
	Editable           bool     `json:"editable"`
	Size               int      `json:"size"`
	Mode               string   `json:"mode"`
	IsExternal         bool     `json:"is_external"`
	ExternalType       string   `json:"external_type"`
	IsPublic           bool     `json:"is_public"`
	PublicURLShared    bool     `json:"public_url_shared"`
	DisplayAsBot       bool     `json:"display_as_bot"`
	Username           string   `json:"username"`
	URLPrivate         string   `json:"url_private"`
	URLPrivateDownload string   `json:"url_private_download"`
	Thumb64            string   `json:"thumb_64"`
	Thumb80            string   `json:"thumb_80"`
	Thumb360           string   `json:"thumb_360"`
	Thumb360W          int      `json:"thumb_360_w"`
	Thumb360H          int      `json:"thumb_360_h"`
	Thumb480           string   `json:"thumb_480,omitempty"`
	Thumb480W          int      `json:"thumb_480_w,omitempty"`
	Thumb480H          int      `json:"thumb_480_h,omitempty"`
	Thumb160           string   `json:"thumb_160"`
	Thumb720           string   `json:"thumb_720,omitempty"`
	Thumb720W          int      `json:"thumb_720_w,omitempty"`
	Thumb720H          int      `json:"thumb_720_h,omitempty"`
	Thumb800           string   `json:"thumb_800,omitempty"`
	Thumb800W          int      `json:"thumb_800_w,omitempty"`
	Thumb800H          int      `json:"thumb_800_h,omitempty"`
	Thumb960           string   `json:"thumb_960,omitempty"`
	Thumb960W          int      `json:"thumb_960_w,omitempty"`
	Thumb960H          int      `json:"thumb_960_h,omitempty"`
	Thumb1024          string   `json:"thumb_1024,omitempty"`
	Thumb1024W         int      `json:"thumb_1024_w,omitempty"`
	Thumb1024H         int      `json:"thumb_1024_h,omitempty"`
	ImageExifRotation  int      `json:"image_exif_rotation"`
	OriginalW          int      `json:"original_w"`
	OriginalH          int      `json:"original_h"`
	Permalink          string   `json:"permalink"`
	PermalinkPublic    string   `json:"permalink_public"`
	Channels           []string `json:"channels"`
	Groups             []any    `json:"groups"`
	Ims                []any    `json:"ims"`
	CommentsCount      int      `json:"comments_count"`
	InitialComment     struct {
		ID        string `json:"id"`
		Created   int    `json:"created"`
		Timestamp int    `json:"timestamp"`
		User      string `json:"user"`
		IsIntro   bool   `json:"is_intro"`
		Comment   string `json:"comment"`
	} `json:"initial_comment,omitempty"`
	Reactions []struct {
		Name  string   `json:"name"`
		Users []string `json:"users"`
		Count int      `json:"count"`
	} `json:"reactions,omitempty"`
}
