package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/matrix-org/gomatrix"
	"github.com/pelletier/go-toml"
)

func main() {
	cfgTree, err := toml.LoadFile("config.toml")
	if err != nil {
		log.Fatalln("Error reading config.toml", err)
	}
	homeserver := cfgTree.Get("homeserver").(string)
	user := cfgTree.Get("user").(string)
	pass := cfgTree.Get("pass").(string)
	room := cfgTree.Get("room").(string)

	cli, err := gomatrix.NewClient(homeserver, "", "")
	if err != nil {
		log.Fatalln("NewClient error:", err)
	}

	login, err := cli.Login(&gomatrix.ReqLogin{
		Type:     "m.login.password",
		User:     user,
		Password: pass,
	})
	if err != nil {
		log.Fatalln("Error logging in:", err)
	}

	cli.SetCredentials(login.UserID, login.AccessToken)

	if _, err := cli.JoinRoom(room, "", nil); err != nil {
		log.Fatalln("JoinRoom error:", err)
	}

	/*
		sendEvent, err := cli.SendText(room, "OMG!")
		if err != nil {
			log.Fatalln("SendText error:", err)
		}
		log.Println(sendEvent.EventID)
	*/

	log.Println("Now connected and listening for '!jasper' in specified room!")

	syncer := cli.Syncer.(*gomatrix.DefaultSyncer)
	syncer.OnEventType("m.room.message", func(ev *gomatrix.Event) {
		msg, ok := ev.Body()
		if ok {
			//fmt.Println("Message: ", msg)
			if strings.HasPrefix(msg, "!jasper") {
				log.Println(ev.Sender, "requested", msg)
				memeText := strings.TrimPrefix(msg, "!jasper ")
				// Grab image from thatsapaddl.in
				tap, err := http.Get("https://thatsapaddl.in/" + memeText)
				if err != nil {
					log.Fatalln("Error getting meme.", err)
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

				imageEvent, err := cli.SendImage(room, "That's a Paddlin'", upload.ContentURI)
				if err != nil {
					log.Fatalln("SendImage error:", err)
				}
				log.Println(imageEvent.EventID)
			}
		}
	})

	// Blocking version
	if err := cli.Sync(); err != nil {
		fmt.Println("Sync() returned ", err)
	}
}
