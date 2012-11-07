
# 金山快盘 (kuaipan) Go SDK

See <http://www.kuaipan.cn/developers/document.htm>

## 初始授权

* 第一步：得到临时 KEY，存在 `Kpan.Key` 中。

		kp := &kpan.Kpan{
			Token: kpan.Token{
				ConsumerKey:    "<YOUR CONSUMER KEY>",
				ConsumerSecret: "<YOUR CONSUMER SECRET>",
			},
		}

		kp.Request("")

* 第二步: 根据第一步得到的 Kpan.Key 访问：

		https://www.kuaipan.cn/api.php?ac=open&op=authorise&oauth_token=<YOUR-KEY>

  进行用户授权。

* 第三步: 利用验证码得到 KEY，存在 `Kpan.Token` 中。

		kp.Access(verifier)

## 使用

授权后就可以使用API了。

**首先要初始化 Kpan：**

	var Kpan = &kpan.Kpan{
		Token: kpan.Token{
			ConsumerKey:    "<YOUR CONSUMER KEY>",
			ConsumerSecret: "<YOUR CONSUMER SECRET>",
			Key:            "<YOUR KEY>",
			Secret:         "<YOUR SECRET>",
		},
		Root:       "app_folder",
	}

**上传文件：**

	Kpan.Upload("/123.txt", []byte{"123456"}, true)

则在快盘中增加了文件 123.txt, 内容为 123456.

**下载文件：**

	data, err := Kpan.Download("/123.txt")

下载刚才上传的文件，data 内容则为 123456。

