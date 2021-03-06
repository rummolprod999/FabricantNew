package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

var UrlXml string

type myjar struct {
	jar map[string][]*http.Cookie
}

func (p *myjar) SetCookies(u *url.URL, cookies []*http.Cookie) {

	p.jar[u.Host] = cookies
}

func (p *myjar) Cookies(u *url.URL) []*http.Cookie {

	return p.jar[u.Host]
}

func GetPage(url string) string {
	defer SaveStack()
	timeout := time.Duration(600 * time.Second)
	var s string
	client := &http.Client{Timeout: timeout}
	jar := &myjar{}
	jar.jar = make(map[string][]*http.Cookie)
	client.Jar = jar
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		Logging(err)
		return s
	}
	req.SetBasicAuth(User, Pass)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko)")

	resp, err := client.Do(req)
	if err != nil {
		Logging(err)
		return s
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Logging(err)
		return s
	}
	return string(body)
}

func DownloadPage(url string) string {
	count := 0
	var st string
	for {
		//fmt.Println("Start download file")
		if count > 5 {
			Logging(fmt.Sprintf("Не скачали файл за %d попыток %s", count, url))
			return st
		}
		st = GetPage(url)
		if st != "" && len(st) > 220 {
			return st
		}
		Logging("Gets empty string", url)
		count++
		time.Sleep(time.Second * 5)
	}
	return st
}
