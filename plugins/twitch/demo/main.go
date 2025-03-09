package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/nicklaw5/helix"
	"io"
	"log"
	"net/http"
	"os"
)

var (
	secret = flag.String("secret", "", "Set client Secret Name")
	cid    = flag.String("cid", "", "Set Client ID Name")
)

func main() {
	flag.Parse()
	if *secret == "" || *cid == "" {
		flag.Usage()
		os.Exit(1)
	}
	client, err := helix.NewClient(&helix.Options{
		ClientID:     *cid,
		ClientSecret: *secret,
	})
	if err != nil {
		log.Printf("Login error: %v", err)
		return
	}

	access, err := client.RequestAppAccessToken([]string{"user:read:email"})
	if err != nil {
		log.Printf("Login error: %v", err)
		return
	}

	fmt.Printf("%+v\n", access)

	// Set the access token on the client
	client.SetAppAccessToken(access.Data.AccessToken)

	users, err := client.GetUsers(&helix.UsersParams{
		Logins: []string{"drseabass"},
	})
	if err != nil {
		log.Printf("Error getting users: %v", err)
		return
	}

	if users.Error != "" {
		log.Printf("Users error: %s", users.Error)
		return
	}

	log.Printf("drseabass: %+v", users.Data.Users[0])
	return

	resp, err := client.CreateEventSubSubscription(&helix.EventSubSubscription{
		Type:    helix.EventSubTypeStreamOnline,
		Version: "1",
		Condition: helix.EventSubCondition{
			BroadcasterUserID: users.Data.Users[0].ID,
		},
		Transport: helix.EventSubTransport{
			Method:   "webhook",
			Callback: "https://rathaus.chrissexton.org/live",
			Secret:   "s3cre7w0rd",
		},
	})
	if err != nil {
		log.Printf("Eventsub error: %v", err)
		return
	}

	fmt.Printf("%+v\n", resp)

	resp, err = client.CreateEventSubSubscription(&helix.EventSubSubscription{
		Type:    helix.EventSubTypeStreamOffline,
		Version: "1",
		Condition: helix.EventSubCondition{
			BroadcasterUserID: users.Data.Users[0].ID,
		},
		Transport: helix.EventSubTransport{
			Method:   "webhook",
			Callback: "https://rathaus.chrissexton.org/offline",
			Secret:   "s3cre7w0rd",
		},
	})
	if err != nil {
		log.Printf("Eventsub error: %v", err)
		return
	}

	fmt.Printf("%+v\n", resp)

	http.HandleFunc("/offline", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			return
		}
		defer r.Body.Close()
		// verify that the notification came from twitch using the secret.
		if !helix.VerifyEventSubNotification("s3cre7w0rd", r.Header, string(body)) {
			log.Println("no valid signature on subscription")
			return
		} else {
			log.Println("verified signature for subscription")
		}
		var vals map[string]any
		if err = json.Unmarshal(body, &vals); err != nil {
			log.Println(err)
			return
		}

		if challenge, ok := vals["challenge"]; ok {
			w.Write([]byte(challenge.(string)))
			return
		}

		log.Printf("got offline webhook: %v\n", vals)
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	http.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			return
		}
		defer r.Body.Close()
		// verify that the notification came from twitch using the secret.
		if !helix.VerifyEventSubNotification("s3cre7w0rd", r.Header, string(body)) {
			log.Println("no valid signature on subscription")
			return
		} else {
			log.Println("verified signature for subscription")
		}
		var vals map[string]any
		if err = json.Unmarshal(body, &vals); err != nil {
			log.Println(err)
			return
		}

		if challenge, ok := vals["challenge"]; ok {
			w.Write([]byte(challenge.(string)))
			return
		}

		log.Printf("got live webhook: %v\n", vals)
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	http.ListenAndServe("0.0.0.0:1337", nil)
}
