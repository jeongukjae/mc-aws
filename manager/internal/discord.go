package internal

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type discordWebhookPayload struct {
	Content string `json:"content"`
}

func SubscribeForWebhook(reader *bufio.Reader, webhookUrl string, quit <-chan bool) <-chan bool {
	isDone := make(chan bool)

	go (func(reader *bufio.Reader, webhookUrl string) {
		size := 2000
		buf := make([]byte, size)

		for {
			time.Sleep(time.Second)
			select {
			case <-quit:
				isDone <- true
				return
			default:
				nr, err := reader.Read(buf)
				if nr > 0 {
					l := string(buf[0:nr])
					log.Print("MCLOG]", l)
					fireDiscordWebhook(l, webhookUrl)
				}
				if err != nil {
					if err == io.EOF {
						continue
					}
					log.Println(err)
				}
			}
		}
	})(reader, webhookUrl)

	return isDone
}

func fireDiscordWebhook(l string, webhookUrl string) {
	payload, err := json.Marshal(discordWebhookPayload{Content: l})
	if err != nil {
		log.Println("Cannot marshal", err)
	}

	resp, err := http.Post(webhookUrl, "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Println(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			log.Println(string(respBody))
		} else {
			log.Println("Sending request failed")
		}
	}
}
