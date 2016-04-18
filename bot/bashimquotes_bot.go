package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	elastigo "github.com/mattbaird/elastigo/lib"
)

const (
	elasticIndice = "bashimquotes"
)

func checkErr(err error) {
	if err != nil {
		log.Println(err.Error())
	}
}

type Quote struct {
	ID   int
	Body string
	URL  string
}

func searchQuote(text string) Quote {
	conn := elastigo.NewConn()
	conn.Domain = os.Getenv("ELASTICSEARCH_HOST")
	defer conn.Close()

	searchJson := fmt.Sprintf(`{
        "query" : {
            "match" : {
                "Body" : "%s"
            }
        }
    }`, text)

	log.Println("Search started")
	out, err := conn.Search(elasticIndice, "quote", nil, searchJson)
	checkErr(err)

	var quote Quote
	if len(out.Hits.Hits) >= 1 {
		log.Printf("Found %d quotes", len(out.Hits.Hits))

		rand.Seed(time.Now().UTC().UnixNano())
		i := rand.Intn(len(out.Hits.Hits))

		log.Printf("Display quota # %d", i)
		quoteJSON, err := out.Hits.Hits[i].Source.MarshalJSON()
		checkErr(err)

		json.Unmarshal(quoteJSON, &quote)

	} else {
		log.Println("Didn't find any relations")
	}
	return quote
}

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOTAPI_TOKEN"))
	checkErr(err)

	log.Printf("Auth on account %s", bot.Self.UserName)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates, err := bot.GetUpdatesChan(updateConfig)
	checkErr(err)

	for update := range updates {
		go func() {
			log.Printf("[%s, %s] %s", update.Message.From.FirstName, update.Message.From.LastName, update.Message.Text)
			quote := searchQuote(strings.Replace(update.Message.Text, "\n", " ", -1))
			if quote.Body != "" {
				var msgText string
				if quote.URL != "" {
					msgText = quote.URL + "\n\n"
				}
				msgText = msgText + strings.Replace(html.UnescapeString(quote.Body), "<br />", "\n", -1)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
				// msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
			}
		}()
	}
}
