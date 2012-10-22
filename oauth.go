// Copyright 2012 by J5ive. All rights reserved.
// Use of this source code is governed by BSD license.
//
package kpan

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"errors"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
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

// GET api operation
func (t *Token) ApiGet(uri string, params map[string]string, obj interface{}) error {
	return httpGet(t.MakeUrl("GET", uri, params), obj)
}

// GET api operation for download.
func (t *Token) ApiGetFile(uri string, params map[string]string, w io.Writer) error {
	return httpGetFile(t.MakeUrl("GET", uri, params), w)
}

// GET api operation for download.
func (t *Token) ApiGetBytes(uri string, params map[string]string) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, bytes.MinRead))
	if err := httpGetFile(t.MakeUrl("GET", uri, params), buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (t *Token) MakeUrl(httpMethod, uri string, params map[string]string) string {
	if params == nil {
		params = make(map[string]string)
	}
	params["oauth_signature_method"] = "HMAC-SHA1"
	params["oauth_version"] = "1.0"
	params["oauth_consumer_key"] = t.ConsumerKey
	params["oauth_timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
	params["oauth_nonce"] = nonce()
	if t.Key != "" {
		params["oauth_token"] = t.Key
	}

	query := encodeParams(params)
	return uri +"?"+query+"&oauth_signature="+
		urlEncode(t.sign(httpMethod, uri, query))
}

// For oauth_nonce
func nonce() string {
	return strconv.FormatInt(rand.Int63(), 10)
}

// oauth signature
func (t *Token) sign(httpMethod, uri, params string) string {
	base := httpMethod + "&" +
		urlEncode(uri) + "&" +
		urlEncode(params)
	key := urlEncode(t.ConsumerSecret) + "&" + urlEncode(t.Secret)
	hash := hmac.New(sha1.New, []byte(key))
	hash.Write([]byte(base))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

func encodeParams(params map[string]string) string {
	pairs := make([]string, len(params))
	i := 0
	for k, v := range params {
		pairs[i] = urlEncode(k) + "=" + urlEncode(v)
		i++
	}

	sort.Strings(pairs)
	return strings.Join(pairs, "&")
}


func httpGet(uri string, obj interface{}) error {
	resp, err := http.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return readFormBody(resp, obj)
}

func httpDo(req *http.Request, obj interface{}) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return readFormBody(resp, obj)
}


func readFormBody(resp *http.Response, obj interface{}) (err error) {
	if resp.StatusCode == 200 {
		if obj == nil {
			io.Copy(ioutil.Discard, resp.Body)
		} else {
			err = json.NewDecoder(resp.Body).Decode(obj)
		}
	} else {
		msg := new(ErrorMsg)
		if err = json.NewDecoder(resp.Body).Decode(msg); err == nil {
			err = msg
		} else {
			err = errors.New(resp.Status)
		}
	}

	return
}


// for download api
type DownClient struct {
	http.Client
	cookies []*http.Cookie
}

func NewClient() *http.Client {
	c := &DownClient{}
	c.Jar = c
	return &c.Client
}

func (c *DownClient) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.cookies = cookies
}

func (c *DownClient) Cookies(u *url.URL) []*http.Cookie {
	return c.cookies
}

func httpGetFile(uri string, w io.Writer) error {
	client := DownClient{}
	client.Jar = &client
	res, err := client.Get(uri)
	if err == nil {
		defer res.Body.Close()
		if res.StatusCode == 200 {
			_, err = io.Copy(w, res.Body)
		} else {
			err = readFormBody(res, nil)
		}
	}
	return err
}

const xdigit = "0123456789ABCDEF"

// urlEncode percent-encodes a string as defined in RFC 3986.
// 注意：对空格处理与 url.QueryEscape 不同
func urlEncode(s string) string {
	n := encodeCount(s)
	if n == len(s) {
		return s
	}
	b := make([]byte, 0, n)
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

func encodeCount(s string) (n int) {
	for i := 0; i < len(s); i++ {
		if isEncodable(s[i]) {
			n += 3
		} else {
			n++
		}
	}
	return
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

