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

const n = 1000

type entry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// 生成一个指定长度的随机字符串
func randomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// POST JSON data
func postJSON(url string, data []byte) string {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err = res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// fmt.Println(res.Status)
	// fmt.Println(res.Header)
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	return string(body)
}

func main() {
	entries := make([]entry, n)
	for i := 0; i < n; i++ {
		key := randomString(10)
		value := randomString(15)
		entries[i] = entry{Key: key, Value: value}
	}

	data, err := json.Marshal(entries)
	if err != nil {
		log.Fatal("json.marshal failed, err:", err)
	}
	res := postJSON("http://127.0.0.1:6201/entries", data)
	fmt.Println(res)
}
