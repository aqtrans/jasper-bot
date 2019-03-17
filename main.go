package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/matrix-org/gomatrix"
	"github.com/pelletier/go-toml"
)

type config struct {
	Homeserver string
	User       string
	Pass       string
	Name       string
}

func main() {
	confFile := flag.String("conf", "config.toml", "Path to the TOML config file.")
	flag.Parse()

	cfgTree, err := toml.LoadFile(*confFile)
	if err != nil {
		log.Fatalln("Error reading", *confFile, err)
	}
	var cfg config
	err = cfgTree.Unmarshal(&cfg)
	if err != nil {
		log.Fatalln("Error unmarshaling config:", err)
	}
	/*
		if !cfgTree.Has("homeserver") {
			log.Fatalln(*confFile, "does not have homeserver defined.")
		}
		if !cfgTree.Has("user") {
			log.Fatalln(*confFile, "does not have user defined.")
		}
		if !cfgTree.Has("pass") {
			log.Fatalln(*confFile, "does not have pass defined.")
		}


		homeserver := cfgTree.Get("homeserver").(string)
		user := cfgTree.Get("user").(string)
		pass := cfgTree.Get("pass").(string)
		displayName := cfgTree.Get("displayName").(string)
	*/
	log.Println(cfg)

	cli, err := gomatrix.NewClient(cfg.Homeserver, "", "")
	if err != nil {
		log.Fatalln("NewClient error:", err)
	}

	login, err := cli.Login(&gomatrix.ReqLogin{
		Type:     "m.login.password",
		User:     cfg.User,
		Password: cfg.Pass,
	})
	if err != nil {
		log.Fatalln("Error logging in:", err)
	}

	log.Println("Access Token:", login.AccessToken)

	cli.SetCredentials(login.UserID, login.AccessToken)
	cli.SetDisplayName(cfg.Name)

	/*
		sendEvent, err := cli.SendText(room, "OMG!")
		if err != nil {
			log.Fatalln("SendText error:", err)
		}
		log.Println(sendEvent.EventID)
	*/

	log.Println("Now connected. Invite", login.UserID, "to any rooms and start sending it messages starting with !jasper.")

	syncer := cli.Syncer.(*gomatrix.DefaultSyncer)
	// Listen for room invites:
	syncer.OnEventType("m.room.member", func(ev *gomatrix.Event) {
		if *ev.StateKey != login.UserID {
			return // not our member event
		}
		m := ev.Content["membership"]
		membership, ok := m.(string)
		if !ok {
			return
		}
		if membership == "invite" {
			log.Println("Invite received:", ev.Sender, ev)
			if _, err := cli.JoinRoom(ev.RoomID, "", nil); err != nil {
				log.Fatalln("JoinRoom error:", err)
			}
		}
	})
	syncer.OnEventType("m.room.message", func(ev *gomatrix.Event) {
		msg, ok := ev.Body()
		if ok {
			// Jasper / Thatsapaddl.in handling:
			if strings.HasPrefix(msg, "!jasper") {
				log.Println(ev.Sender, "requested", msg)
				memeText := strings.TrimPrefix(msg, "!jasper ")
				// Grab image from thatsapaddl.in
				tap, err := http.Get("https://thatsapaddl.in/" + memeText)
				if err != nil {
					log.Fatalln("Error getting meme:", err)
				}
				tapImage := new(bytes.Buffer)
				_, err = io.Copy(tapImage, tap.Body)
				tap.Body.Close()

				/*
					encodedPNG := new(bytes.Buffer)
					newimage := paddle("Ditching Hangouts")
					err = png.Encode(encodedPNG, newimage)
					if err != nil {
						log.Println(err)
					}
				*/

				upload, err := cli.UploadToContentRepo(tapImage, "image/png", int64(tapImage.Len()))
				if err != nil {
					log.Fatalln("Error uploading meme:", err)
				}

				_, err = cli.SendImage(ev.RoomID, "That's a Paddlin'", upload.ContentURI)
				if err != nil {
					log.Fatalln("SendImage error:", err)
				}
			}
			// Squanch.Space handling:
			re := regexp.MustCompile(`^;([A-z0-9_.])+\b`)
			if re.MatchString(msg) {
				memeText := strings.TrimPrefix(msg, ";")
				log.Println(ev.Sender, " squanched ", memeText)
				imgURL := "https://i.squanch.space/" + memeText
				resp, err := http.Get(imgURL)
				if err != nil {
					log.Fatalln("error getting meme from squanch.space:", err)
				}
				// This means meme should exist
				if resp.StatusCode == 200 {
					resUpload, err := cli.UploadLink(imgURL)
					if err != nil {
						log.Fatalln("error uploading squanch.space meme:", err)
					}
					_, err = cli.SendImage(ev.RoomID, memeText, resUpload.ContentURI)
					if err != nil {
						log.Fatalln("SendImage error:", err)
					}
				}
			}
		}
	})

	// Blocking version
	if err := cli.Sync(); err != nil {
		fmt.Println("Sync() returned ", err)
	}
}
