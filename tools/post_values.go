// 生成一些测试数据并且保存到Tomahawk集群中

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type entry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// 生成一个指定长度的随机字符串
func randomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// 生成随机的中文汉字
func randomChineseWords(num int) string {
	randInt := func(min, max int64) int64 {
		rand.Seed(time.Now().UnixNano())
		return min + rand.Int63n(max-min)
	}

	words := make([]rune, num)
	for i := range words {
		words[i] = rune(randInt(19968, 40869))
	}
	return string(words)
}

// POST JSON data
func postJSON(url string, data []byte) string {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Panic(err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		err = res.Body.Close()
		if err != nil {
			log.Panic(err)
		}
	}()

	// fmt.Println(res.Status)
	// fmt.Println(res.Header)
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Panic(err)
	}
	return string(body)
}

func main() {
	entries := make([]entry, 0)
	entries = append(entries, entry{Key: "😊😊😊", Value: "😁😁😁😁😁😆😆😆😆😆"})

	for i := 0; i < 10000; i++ {
		key := randomString(10)
		value := randomString(15)
		entries = append(entries, entry{Key: key, Value: value})
	}

	for i := 0; i < 200; i++ {
		key := randomChineseWords(5)
		value := randomChineseWords(12)
		entries = append(entries, entry{Key: key, Value: value})
	}

	data, err := json.Marshal(entries)
	if err != nil {
		log.Panic("json.marshal failed, err:", err)
	}
	res := postJSON("http://172.21.3.92:6200/entries", data)
	fmt.Println(res)
}
