// Copyright 2012 by J5ive. All rights reserved.
// Use of this source code is governed by BSD license.
//
// 金山快盘 (kuaipan) Go SDK
//
// See http://www.kuaipan.cn/developers/document.htm
//
package kpan

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strconv"
)

type Kpan struct {
	Token
	UserId     int64
	ChargedDir string
	Root       string // app_folder or kuaipan
	uri        string // upload url
}

// 进行 oauth 认证的第一步.
// 第二步需要到 https://www.kuaipan.cn/api.php?ac=open&op=authorise&oauth_token=<Kpan.Key> 进行验证。
// 验证完得到验证码。
func (p *Kpan) Request(callback string) (callback_confirmed bool, err error) {
	params := make(map[string]string)
	if callback != "" {
		params["oauth_callback"] = callback
	}
	data := make(map[string]interface{})
	err = p.ApiGet(
		"https://openapi.kuaipan.cn/open/requestToken",
		params,
		&data)
	if err == nil {
		p.Key = data["oauth_token"].(string)
		p.Secret = data["oauth_token_secret"].(string)
		callback_confirmed = data["oauth_callback_confirmed"].(bool)
	}
	return
}

// 进行 oauth 认证的第三步.
// 利用验证码得到 KEY. 如果已授权, 验证码也可为空。
func (p *Kpan) Access(verifier string) error {
	data := make(map[string]interface{})
	params := make(map[string]string)
	if verifier != "" {
		params["oauth_verifier"] = verifier
	}

	err := p.ApiGet(
		"https://openapi.kuaipan.cn/open/accessToken",
		params,
		&data)
	if err == nil {
		p.Key = data["oauth_token"].(string)
		p.Secret = data["oauth_token_secret"].(string)
		p.UserId = int64(data["user_id"].(float64))
		p.ChargedDir = data["charged_dir"].(string)
	}
	return err
}


type AccountInfo struct {
	UserId      int    `json:"user_id"`
	UserName    string `json:"user_name"`
	MaxFileSize int    `json:"max_file_size"`
	QuotaTotal  int64  `json:"quota_total"`
	QuotaUsed   int64  `json:"quota_used"`

	QuotoRecycled int64 `json:"quota_recycled"`
}

// 查看用户信息
func (p *Kpan) AccountInfo() (*AccountInfo, error) {
	info := &AccountInfo{ QuotoRecycled: -1 }
	err := p.ApiGet("http://openapi.kuaipan.cn/1/account_info", nil, info)
	return info, err
}


type DirInfo struct {
	Path string `json:"path"`
	Root string `json:"root"`

	Hash       string     `json:"hash"`
	FileId     string     `json:"file_id"`
	Type       string     `json:"type"`
	Size       int        `json:"size"`
	CreateTime string     `json:"create_time"`
	ModifyTime string     `json:"modify_time"`
	Name       string     `json:"name"`
	Rev        string     `json:"rev"`
	IsDeleted  bool       `json:"is_deleted"`
	Files      []FileInfo `json:"files"`
}

type FileInfo struct {
	FileId     string `json:"file_id"`
	Type       string `json:"type"`
	Size       int    `json:"size"`
	CreateTime string `json:"create_time"`
	ModifyTime string `json:"modify_time"`
	Name       string `json:"name"`
	IsDeleted  bool   `json:"is_deleted"`

	Rev string `json:"rev"`
}

// 获取单个文件，文件夹信息
// params: list, file_limit, page, page_size, filter_ext, sort_by
func (p *Kpan) Metadata(pathname string, params map[string]string) (*DirInfo, error) {
	info := &DirInfo{ Size: -1 }
	err := p.ApiGet(
		join("http://openapi.kuaipan.cn/1/metadata/" + p.Root, pathname),
		params,
		info)
	return info, err
}


type ShareInfo struct {
	Url        string `json:"url"`
	AccessCode string `json:"access_code"`
}

// 创建并获取一个文件的分享链接
func (p *Kpan) Share(pathname, displayName, accessCode string) (*ShareInfo, error) {
	params := map[string]string{}
	if displayName != "" {
		params["name"] = displayName
	}
	if accessCode != "" {
		params["access_code"] = accessCode
	}
	res := new(ShareInfo)
	err := p.ApiGet(
		join("http://openapi.kuaipan.cn/1/shares/" + p.Root, pathname),
		params,
		res)
	return res, err
}


type CreateResult struct {
	FileId string `json:"file_id"`
	Path   string `json:"path"`
	Root   string `json:"root"`
}

// 新建文件夹
func (p *Kpan) CreateFolder(pathname string) (*CreateResult, error) {
	res := new(CreateResult)
	err := p.ApiGet(
		"http://openapi.kuaipan.cn/1/fileops/create_folder",
		map[string]string{"path": pathname, "root": p.Root},
		res)
	return res, err
}

// 删除文件，文件夹，以及文件夹下所有文件到回收站
func (p *Kpan) Delete(pathname string, toRecycle bool) error {
	return p.ApiGet(
		"http://openapi.kuaipan.cn/1/fileops/delete",
		map[string]string{
			"path":       pathname,
			"root":       p.Root,
			"to_recycle": strconv.FormatBool(toRecycle),
		},
		nil)
}

// 移动文件，文件夹
func (p *Kpan) Move(fromPath, toPath string) error {
	return p.ApiGet(
		"http://openapi.kuaipan.cn/1/fileops/move",
		map[string]string{
			"from_path": fromPath,
			"to_path":   toPath,
			"root":      p.Root,
		},
		nil)
}

// 复制文件，文件夹
func (p *Kpan) Copy(fromPath, toPath, copyRef string) error {
	params := map[string]string{
		"to_path": toPath,
		"root":    p.Root,
	}
	if fromPath != "" {
		params["from_path"] = fromPath
	}
	if copyRef != "" {
		params["copy_ref"] = copyRef
	}
	return p.ApiGet("http://openapi.kuaipan.cn/1/fileops/copy", params, nil)
}


type CopyRefResult struct {
	CopyRef string `json:"copy_ref"`
	Expires string `json:"expires"`
}

// 产生一个复制引用（ref）
func (p *Kpan) CopyRef(pathname string) (*CopyRefResult, error) {
	res := new(CopyRefResult)
	err := p.ApiGet(
		join("http://openapi.kuaipan.cn/1/copy_ref/" + p.Root, pathname),
		nil,
		res)
	return res, err
}


// 下载
// TODO: HTTP Range Retrieval Requests
func (p *Kpan) Download(pathname string) ([]byte, error) {
	return p.ApiGetBytes(
		"http://api-content.dfs.kuaipan.cn/1/fileops/download_file",
		map[string]string{
			"path": pathname,
			"root": p.Root,
		})
}

// 下载
func (p *Kpan) DownloadTo(pathname string, w io.Writer) error {
	return p.ApiGetFile(
		"http://api-content.dfs.kuaipan.cn/1/fileops/download_file",
		map[string]string{
			"path": pathname,
			"root": p.Root,
		},
		w)
}

// 下载文件并保存到本地
func (p *Kpan) DownloadFile(remoteFile, localFile string) error {
	f, err := os.Create(NameFromTo(remoteFile, localFile))
	if err != nil {
		return err
	}
	defer f.Close()
	return p.DownloadTo(remoteFile, f)
}

func NameFromTo(from, to string) string {
	if len(to) == 0 {
		to = path.Base(from)
	} else if to[len(to)-1] == '/' {
		to = path.Join(to, path.Base(from))
	}
	return to
}

// 获取缩略图
func (p *Kpan) Thumnail(pathname string, width, height int, w io.Writer) ([]byte, error) {
	return p.ApiGetBytes(
		"http://conv.kuaipan.cn/1/fileops/thumbnail",
		map[string]string{
			"path":  pathname,
			"root":  p.Root,
			"width": strconv.Itoa(width),
			"height": strconv.Itoa(height),
		})
}

// 文档转换
func (p *Kpan) DocumentView(pathname, typ, view string) ([]byte, error) {
	return p.ApiGetBytes(
		"http://conv.kuaipan.cn/1/fileops/documentView",
		map[string]string{
			"path": pathname,
			"root": p.Root,
			"type": typ,
			"view": view,
		})
}

type upLocate struct {
	Url string `json:"url"`
}

// 获取上传url (1st step of openapi)
func (p *Kpan) UploadLocate() (string, error) {
	var res upLocate
	err := p.ApiGet(
		"http://api-content.dfs.kuaipan.cn/1/fileops/upload_locate",
		nil, &res)
	return join(res.Url, "/1/fileops/upload_file"), err
}

func join(host, pathname string) string {
	if len(host) == 0 || len(pathname) == 0 {
		return host + pathname
	}
	if host[len(host)-1] == '/' {
		if pathname[0] == '/' {
			return host + pathname[1:]
		} else {
			return host + pathname
		}
	}
	if pathname[0] == '/' {
		return host + pathname
	}
	return host + "/" + pathname
}

type UploadResult struct {
	FileId     string `json:"file_id"`
	Type       string `json:"type"`
	Rev        string `json:"rev"`
	Size       int    `json:"size,string"`	// 文档中是int, 但实际返回 string
	// Stat string  `json:"stat"`	// 文档中无, 成功返回 OK
	// Url string  `json:"url"`		// 文档中无

	CreateTime string `json:"create_time"`
	ModifyTime string `json:"modify_time"`
	IsDeleted  bool   `json:"is_deleted"`
}

// 根据 UploadLocate 得到的 url 上传 (2nd step of openapi)
// 必须提供size, 否则返回411错误。
func (p *Kpan) DoUpload(uri, pathname string, r io.Reader, size int, overwrite bool) (res *UploadResult, err error) {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	w.CreateFormFile("file", pathname)

	uri = p.MakeUrl("POST", uri, map[string]string{
		"overwrite": btoa(overwrite),
		"root":      p.Root,
		"path":      pathname,
	})
	req, _ := http.NewRequest("POST", uri, io.MultiReader(buf, r, &partEnder{buf, w, false}))
	req.ContentLength = int64(buf.Len() + size + len(w.Boundary()) + 8)
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Content-Type", w.FormDataContentType())
//	req.Header.Set("Connection", "Close")
	req.Header.Set("User-Agent", "kpancli")

	res = new(UploadResult)
	err = httpDo(req, res)
	return
}

func btoa(b bool) string {
	if b {
		return "True"
	}
	return "False"
}


type partEnder struct {
	buf *bytes.Buffer
	mw *multipart.Writer
	closed bool
}

func (r *partEnder) Read(b []byte) (n int, err error) {
	if r.closed {
		return 0, io.EOF
	}
	r.closed = true
	r.mw.Close()
	return r.buf.Read(b)
}



// 上传
// 必须提供size, 否则返回411错误。
func (p *Kpan) UploadFrom(pathname string, r io.Reader, size int, overwrite bool) (res *UploadResult, err error) {
	if p.uri == "" {
		p.uri, err = p.UploadLocate()
	}
	if err == nil {
		res, err = p.DoUpload(p.uri, pathname, r, size, overwrite)
	}
	return
}

// 上传
func (p *Kpan) Upload(pathname string, data []byte, overwrite bool) (res *UploadResult, err error) {
	r := bytes.NewReader(data)
	return p.UploadFrom(pathname, r, len(data), overwrite)
}

// 上传本地文件
func (p *Kpan) UploadFile(remoteFile, localFile string, overwrite bool) (res *UploadResult, err error) {
	f, err := os.Open(localFile)
	if err != nil {
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err == nil {
		remoteFile = NameFromTo(localFile, remoteFile)
		res, err = p.UploadFrom(remoteFile, f, int(fi.Size()), overwrite)
	}
	return
}



