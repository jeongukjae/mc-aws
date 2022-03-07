package internal

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type discordWebhookPayload struct {
	Content string `json:"content"`
}

func SubscribeForWebhook(reader *bufio.Reader, webhookUrl string, quit <-chan bool, botMsg <-chan string) <-chan bool {
	isDone := make(chan bool)
	logChan := make(chan string)

	go (func() {
		size := 2000
		buf := make([]byte, size)

		for {
			nr, _ := reader.Read(buf)
			if nr > 0 {
				l := string(buf[0:nr])
				log.Print("MC]", l)
				logChan <- l
			}
		}
	})()

	go (func() {
		for {
			time.Sleep(time.Second * 3)
			select {
			case <-quit:
				isDone <- true
				return
			case msg := <-botMsg:
				fireDiscordWebhook(msg, webhookUrl)
			case msg := <-logChan:
				fireDiscordWebhook(msg, webhookUrl)
			}
		}
	})()

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
