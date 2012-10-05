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
	"strings"
	"strconv"
	"path"
	"io/ioutil"
	"mime/multipart"
	"net/http"
)

type Kpan struct {
	Token
	UserId     int64
	ChargedDir string
	Root       string // app_folder or kuanpan
	host       string // upload url
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
	err = p.GetJson(
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

	err := p.GetJson(
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

	QuotoRecycled int64 `json:"quota_recycled,omitempty"`
}

// 查看用户信息
func (p *Kpan) AccountInfo() (*AccountInfo, error) {
	info := &AccountInfo{ QuotoRecycled: -1 }
	err := p.GetJson("http://openapi.kuaipan.cn/1/account_info", nil, info)
	return info, err
}


type DirInfo struct {
	Path string `json:"path"`
	Root string `json:"root"`

	//Hash string
	FileId     string     `json:"file_id,omitempty"`
	Type       string     `json:"type,omitempty"`
	Size       int        `json:"size,omitempty"`
	CreateTime string     `json:"create_time,omitempty"`
	ModifyTime string     `json:"modify_time,omitempty"`
	Name       string     `json:"name,omitempty"`
	Rev        string     `json:"rev,omitempty"`
	IsDeleted  bool       `json:"is_deleted,omitempty"`
	Files      []FileInfo `json:"files,omitempty"`
}

type FileInfo struct {
	FileId     string `json:"file_id"`
	Type       string `json:"type"`
	Size       int    `json:"size"`
	CreateTime string `json:"create_time"`
	ModifyTime string `json:"modify_time"`
	Name       string `json:"name"`
	IsDeleted  bool   `json:"is_deleted"`

	Rev string `json:"rev,omitempty"`
}

// 获取单个文件，文件夹信息
// params: list, file_limit, page, page_size, filter_ext, sort_by
func (p *Kpan) Metadata(pathname string, params map[string]string) (*DirInfo, error) {
	info := &DirInfo{ Size: -1 }
	err := p.GetJson(
		"http://openapi.kuaipan.cn/1/metadata/" + p.Root + addSep(pathname),
		params,
		info)
	return info, err
}

func addSep(pathname string) string {
	if !strings.HasPrefix(pathname, "/") {
		pathname = "/" + pathname
	}
	return pathname
}


type ShareInfo struct {
	Url        string `json:"url"`
	AccessCode string `json:"access_code,omitempty"`
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
	err := p.GetJson(
		"http://openapi.kuaipan.cn/1/shares/" + p.Root + addSep(pathname),
		params,
		res)
	return res, err
}


type CreateResult struct {
	FileId string `json:"file_id"`
	Path   string `json:"path,omitempty"`
	Root   string `json:"root,omitempty"`
}

// 新建文件夹
func (p *Kpan) CreateFolder(pathname string) (*CreateResult, error) {
	res := new(CreateResult)
	err := p.GetJson(
		"http://openapi.kuaipan.cn/1/fileops/create_folder",
		map[string]string{"path": pathname, "root": p.Root},
		res)
	return res, err
}

// 删除文件，文件夹，以及文件夹下所有文件到回收站
func (p *Kpan) Delete(pathname string, toRecycle bool) error {
	_, err := p.Get(
		"http://openapi.kuaipan.cn/1/fileops/delete",
		map[string]string{
			"path":       pathname,
			"root":       p.Root,
			"to_recycle": strconv.FormatBool(toRecycle),
		})
	return err
}

// 移动文件，文件夹
func (p *Kpan) Move(fromPath, toPath string) error {
	_, err := p.Get(
		"http://openapi.kuaipan.cn/1/fileops/move",
		map[string]string{
			"from_path": fromPath,
			"to_path":   toPath,
			"root":      p.Root,
		})
	return err
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
	_, err := p.Get("http://openapi.kuaipan.cn/1/fileops/copy", params)
	return err
}


type CopyRefResult struct {
	CopyRef string `json:"copy_ref"`
	Expires string `json:"expires"`
}

// 产生一个复制引用（ref）
func (p *Kpan) CopyRef(pathname string) (*CopyRefResult, error) {
	res := new(CopyRefResult)
	err := p.GetJson(
		"http://openapi.kuaipan.cn/1/copy_ref/" + p.Root + addSep(pathname),
		nil,
		res)
	return res, err
}


// 下载
// TODO: HTTP Range Retrieval Requests
func (p *Kpan) Download(pathname string) ([]byte, error) {
	return p.GetFile(
		"http://api-content.dfs.kuaipan.cn/1/fileops/download_file",
		map[string]string{
			"path": pathname,
			"root": p.Root,
		})
}

// 下载文件并保存到本地
func (p *Kpan) DownloadFile(remoteFile, localFile string) error {
	if len(localFile) == 0 {
		localFile = path.Base(remoteFile)
	} else if localFile[len(localFile)-1] == '/' {
		localFile = path.Join(localFile, path.Base(remoteFile))
	}

	data, err := p.Download(remoteFile)
	if err == nil {
		err = ioutil.WriteFile(localFile, data, 0666)
	}
	return err
}

// 获取缩略图
func (p *Kpan) Thumnail(pathname string, width, height int) ([]byte, error) {
	return p.GetFile(
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
	return p.GetFile(
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
	err := p.GetJson(
		"http://api-content.dfs.kuaipan.cn/1/fileops/upload_locate",
		nil, &res)
	return res.Url, err
}


type UploadResult struct {
	FileId     string `json:"file_id"`
	Type       string `json:"type"`
	Rev        string `json:"rev"`
	Size       int    `json:"size"`
	CreateTime string `json:"create_time,omitempty"`
	ModifyTime string `json:"modify_time,omitempty"`
	IsDeleted  bool   `json:"is_deleted,omitempty"`
}

// 根据 UploadLocate 得到的 host url 上传 (2nd step of openapi)
func (p *Kpan) UploadTo(host, pathname string, overwrite bool, data []byte) (res *UploadResult, err error) {
	res = new(UploadResult)
	err = p.DoJson(
		p.newUploadRequest(host, pathname, data),
		map[string]string{
			"overwrite": btoa(overwrite),
			"root":      p.Root,
			"path":      pathname,
		},
		res)
	return
}

func (p *Kpan) newUploadRequest(host, pathname string, data []byte) *http.Request {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	part, _ := w.CreateFormFile("file", pathname)
	part.Write(data)
	w.Close()

	req, _ := http.NewRequest("POST", host + addSep(pathname), buf)
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Content-Type", w.FormDataContentType())
//	req.Header.Set("Content-Length", strconv.Itoa(buf.Len()))
	req.Header.Set("Connection", "Close")
	req.Header.Set("User-Agent", "kpancli")

	return req
}

func btoa(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

// 上传
func (p *Kpan) Upload(pathname string, overwrite bool, data []byte) (res *UploadResult, err error) {
	if p.host == "" {
		p.host, err = p.UploadLocate()
	}
	if err == nil {
		res, err = p.UploadTo(p.host, pathname, overwrite, data)
	}
	return
}

// 上传本地文件
func (p *Kpan) UploadFile(localFile, remoteFile string, overwrite bool) (*UploadResult, error) {
	if len(remoteFile) == 0 {
		remoteFile = path.Base(localFile)
	} else if remoteFile[len(remoteFile)-1] == '/' {
		remoteFile = path.Join(remoteFile, path.Base(localFile))
	}

	data, err := ioutil.ReadFile(localFile)
	if err != nil {
		return nil, err
	}
	return p.Upload(remoteFile, overwrite, data)
}

