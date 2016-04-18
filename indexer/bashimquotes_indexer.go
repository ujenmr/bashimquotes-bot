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

func main() {
	getQuote := func(id int) Quote {
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
		return Quote{ID: id, Body: fromCP1251toUTF8(text), URL: url}
	}

	elasticIndexing := func(quote Quote) {
		conn := elastigo.NewConn()
		conn.Domain = os.Getenv("ELASTICSEARCH_HOST")
		defer conn.Close()

		quoteJSON, err := json.Marshal(quote)
		checkErr(err)
		log.Println("Encoded to json #" + strconv.Itoa(quote.ID))

		_, err = conn.Index(elasticIndice, "quote", strconv.Itoa(quote.ID), nil, string(quoteJSON))
		checkErr(err)

		log.Println("Added to elastic #" + strconv.Itoa(quote.ID))
	}
	for i := 1; i <= 438894; i++ {
		go elasticIndexing(getQuote(i))
		// time.Sleep(30 * time.Millisecond)
	}
}
