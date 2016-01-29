package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/ChimeraCoder/anaconda"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Tweet is object of tweet.
type Tweet struct {
	ID        string
	Text      string
	ImageURLs []string
	Date      time.Time
	Place     string
	Menu      string
	Price     int
	Feel      string
}

func main() {
	anaconda.SetConsumerKey(os.Getenv("CONSUMER_KEY"))
	anaconda.SetConsumerSecret(os.Getenv("CONSUMER_SECRET"))

	tweets, err := getTweetFromFile()
	if err != nil {
		panic(err)
	}

	tweetsFromAPI, err := getTweetFromAPI(tweets[0].ID)
	if err != nil {
		panic(err)
	}
	tweets = append(tweets, tweetsFromAPI...)

	dump(tweets, "data/habomai.json")
}

func getTweetFromAPI(sinceID string) (tweets []Tweet, err error) {
	api := anaconda.NewTwitterApi(os.Getenv("ACCESS_TOKEN"), os.Getenv("ACCESS_TOKEN_SECRET"))
	v := url.Values{}
	v.Add("screen_name", "habomaijiro")
	v.Add("count", "200")
	if sinceID != "" {
		v.Add("since_id", sinceID)
	}
	tl, err := api.GetUserTimeline(v)
	if err != nil {
		return
	}

	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return
	}
	tweets = make([]Tweet, len(tl))
	for i, v := range tl {
		var t Tweet
		t.ID = v.IdStr
		if parsedT, err := time.Parse(time.RubyDate, v.CreatedAt); err == nil {
			t.Date = parsedT.In(jst)
		}
		t.ImageURLs = make([]string, len(v.Entities.Media))
		for j, m := range v.Entities.Media {
			t.ImageURLs[j] = m.Media_url_https
		}
		analizeText(&t, v.Text)
		tweets[i] = t
	}
	return
}

func getTweetFromFile() (tweets []Tweet, err error) {
	f, err := os.Open("twitter.html")
	if err != nil {
		return
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return
	}

	doc.Find(".js-stream-tweet").Each(func(i int, s *goquery.Selection) {
		if s.HasClass("separated-module") || s.HasClass("has-profile-promoted-tweet") {
			return
		}
		var t Tweet
		analizeText(&t, s.Find(".tweet-text").Text())

		// 日時取得
		if ts, isExist := s.Find(".js-short-timestamp").Attr("data-time"); isExist {
			if fts, err := strconv.ParseInt(ts, 10, 64); err == nil {
				t.Date = time.Unix(fts, 0)
			}
		}

		// 画像取得
		s.Find(".js-adaptive-photo").Each(func(i int, s *goquery.Selection) {
			if url, isExist := s.Attr("data-image-url"); isExist {
				t.ImageURLs = append(t.ImageURLs, url)
			} else {
				fmt.Println("image")
			}
		})

		// ID取得
		t.ID, _ = s.Attr("data-tweet-id")
		if t.ID != "" {
			tweets = append(tweets, t)
		}
	})

	return
}

func analizeText(t *Tweet, text string) (err error) {

	// 本文取得
	t.Text = text

	splitedText := strings.Split(text, "、")

	// 店名取得
	if len(splitedText) >= 2 {
		t.Place = splitedText[1]
	}

	// 価格取得
	if len(splitedText) >= 3 {
		re1, err := regexp.Compile(`\d*YEN`)
		re2, err := regexp.Compile(`YEN`)
		if err != nil {
			return err
		}
		t.Price, err = strconv.Atoi(re2.ReplaceAllString(re1.FindString(splitedText[2]), ""))
		if err != nil {
			return err
		}

		// メニュー取得
		re3, err := regexp.Compile(`.*YEN`)
		if err != nil {
			return err
		}
		t.Menu = re1.ReplaceAllString(re3.FindString(splitedText[2]), "")
	}

	// 感想取得
	re4, err := regexp.Compile(`.*YEN|https?://[\w/:%#\$&\?\(\)~\.=\+\-]+|pic\.twitter\.com.*`)
	t.Feel = strings.TrimSpace(re4.ReplaceAllString(text, ""))

	return
}

func dump(tweets []Tweet, fname string) (err error) {
	b, err := json.MarshalIndent(tweets, "", "\t")
	if err != nil {
		return
	}
	if err = ioutil.WriteFile(fname, b, os.ModePerm); err != nil {
		if err = os.MkdirAll(filepath.Dir(fname), os.ModePerm); err != nil {
			return err
		}
		if err = ioutil.WriteFile(fname, b, os.ModePerm); err != nil {
			return err
		}
	}
	return
}
