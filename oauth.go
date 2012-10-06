// Copyright 2012 by J5ive. All rights reserved.
// Use of this source code is governed by BSD license. 
//
package kpan

import (
	"errors"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
)

func init() {
	rand.Seed(time.Now().Unix())
}


type Token struct {
	ConsumerKey, ConsumerSecret string
	Key, Secret                 string
}

type ErrorMsg struct {
	Msg string `json:"msg"`
}

func (e *ErrorMsg) Error() string {
	return e.Msg
}

// for download api
type DownClient struct {
	http.Client
	cookies []*http.Cookie
}

func (c *DownClient) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.cookies = cookies
}

func (c *DownClient) Cookies(u *url.URL) []*http.Cookie {
	return c.cookies
}

// GET api operation.
func (t *Token) Get(uri string, params map[string]string) ([]byte, error) {
	if params == nil {
		params = make(map[string]string)
	}
	t.Sign("GET", uri, params)
	
	// tr := &http.Transport{
	//    TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	//}
	// client := &http.Client{Transport: tr}
	return t.httpGet(http.DefaultClient, uri, params)
}

// GET api operation for download.
func (t *Token) GetFile(uri string, params map[string]string) ([]byte, error) {
	if params == nil {
		params = make(map[string]string)
	}
	t.Sign("GET", uri, params)
	
	client := DownClient{}
	client.Jar = &client
	return t.httpGet(&client.Client, uri, params)
}

// GET api operation for json object
func (t *Token) GetJson(uri string, params map[string]string, obj interface{}) error {
	if params == nil {
		params = make(map[string]string)
	}
	t.Sign("GET", uri, params)
	return t.httpGetJson(uri, params, obj)
}

// specail api operation, for upload
func (t *Token) DoJson(req *http.Request, params map[string]string, obj interface{}) error {
	if params == nil {
		params = make(map[string]string)
	}
	t.Sign(req.Method, req.URL.String(), params)
	return t.httpDoJson(req, params, obj)
}

// oauth signature
func (t *Token) Sign(httpMethod, uri string, params map[string]string) {
	params["oauth_signature_method"] = "HMAC-SHA1"
	params["oauth_version"] = "1.0"
	params["oauth_consumer_key"] = t.ConsumerKey
	params["oauth_timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
	params["oauth_nonce"] = nonce()
	if t.Key != "" {
		params["oauth_token"] = t.Key
	}

	params["oauth_signature"] = t.getSign(httpMethod, uri, params)
}

func (t *Token) getSign(httpMethod, uri string, params map[string]string) string {
	base := httpMethod + "&" +
		urlEncode(uri) + "&" +
		urlEncode(encodeParams(params, true))

	key := urlEncode(t.ConsumerSecret) + "&" + urlEncode(t.Secret)
	hash := hmac.New(sha1.New, []byte(key))
	hash.Write([]byte(base))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

// For oauth_nonce
func nonce() string {
	return strconv.FormatInt(rand.Int63(), 10)
}

// urlEncode percent-encodes a string as defined in RFC 3986.
func urlEncode(s string) string {
	xdigit := "0123456789ABCDEF"
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isEncodable(c) {
			b = append(b, '%', xdigit[c>>4], xdigit[c&15])
		} else {
			b = append(b, c)
		}
	}
	return string(b)
}

// isEncodable returns true if a given character should be percent-encoded
// according to RFC 3986.
func isEncodable(c byte) bool {
	// return false if c is an unreserved character (see RFC 3986 section 2.3)
	return !((c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' ||
		c == '.' ||
		c == '_' ||
		c == '~')
}


func (t *Token) httpGet(client *http.Client, uri string, params map[string]string) ([]byte, error) {
	res, err := client.Get(uri + "?" + encodeParams(params, false))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if res.StatusCode != 200 {
		if data != nil {
			err = errors.New(string(data))
		} else {
			err = errors.New(res.Status)
		}
	}
	return data, err
}

func (t *Token) httpGetJson(uri string, params map[string]string, obj interface{}) error {
	resp, err := http.Get(uri + "?" + encodeParams(params, false))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return readFormBody(resp, obj)
}

func (t *Token) httpDoJson(req *http.Request, params map[string]string, obj interface{}) error {
	req.URL.RawQuery = encodeParams(params, false)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return readFormBody(resp, obj)
}

func encodeParams(params map[string]string, sorted bool) string {
	pairs := make([]string, len(params))
	i := 0
	for k, v := range params {
		pairs[i] = urlEncode(k) + "=" + urlEncode(v)
		i++
	}

	if sorted {
		sort.Strings(pairs)
	}
	return strings.Join(pairs, "&")
}

func readFormBody(resp *http.Response, obj interface{}) error {
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// println(string(data))

	if resp.StatusCode != 200 {
		msg := new(ErrorMsg)
		if err = json.Unmarshal(data, msg); err == nil {
			err = msg
		} else {
			err = errors.New(resp.Status)
		}
	} else if err == nil {
		err = json.Unmarshal(data, obj)
	}

	return err
}