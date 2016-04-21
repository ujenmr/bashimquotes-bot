package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"

	elastigo "github.com/mattbaird/elastigo/lib"
	_ "golang.org/x/text"
	charmap "golang.org/x/text/encoding/charmap"
)

const (
	elasticIndice = "bashimquotes"
)

type Quote struct {
	ID   int
	Body string
	URL  string
}

func checkErr(err error) {
	if err != nil {
		log.Panic(err.Error())
	}
}

func fromCP1251toUTF8(s string) string {
	dec := charmap.Windows1251.NewDecoder()
	bUTF := make([]byte, len(s)*3)
	n, _, err := dec.Transform(bUTF, []byte(s), false)
	checkErr(err)
	bUTF = bUTF[:n]
	return string(bUTF)
}

func elasticIndexing(quoteChan chan Quote) {
	conn := elastigo.NewConn()
	if os.Getenv("ELASTICSEARCH_HOST") != "" {
		conn.Domain = os.Getenv("ELASTICSEARCH_HOST")
	}
	defer conn.Close()

	for {
		quote := <-quoteChan
		quoteJSON, err := json.Marshal(quote)
		checkErr(err)
		log.Println("Encoded to json #" + strconv.Itoa(quote.ID))

		_, err = conn.Index(elasticIndice, "quote", strconv.Itoa(quote.ID), nil, string(quoteJSON))
		checkErr(err)

		log.Println("Added to elastic #" + strconv.Itoa(quote.ID))
	}
}

func main() {
	getQuote := func(id int, quoteChan chan Quote) {
		url := "http://bash.im/quote/" + strconv.Itoa(id)
		response, err := http.Get(url)
		checkErr(err)
		log.Println("Read message #" + strconv.Itoa(id))

		bytes, err := ioutil.ReadAll(response.Body)
		checkErr(err)

		defer response.Body.Close()

		reg, err := regexp.Compile(`(<div class="text">)(.*)(</div>)`)
		checkErr(err)

		text := reg.FindStringSubmatch(string(bytes))[2]
		log.Println("Parsed message #" + strconv.Itoa(id))
		quoteChan <- Quote{ID: id, Body: fromCP1251toUTF8(text), URL: url}
	}

	var quoteChan chan Quote = make(chan Quote)
	go elasticIndexing(quoteChan)

	for i := 1; i <= 438894; i++ {
		getQuote(i, quoteChan)
	}
}
