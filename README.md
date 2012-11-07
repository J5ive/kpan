
# ��ɽ���� (kuaipan) Go SDK

See <http://www.kuaipan.cn/developers/document.htm>

## ��ʼ��Ȩ

* ��һ�����õ���ʱ KEY������ `Kpan.Key` �С�

		kp := &kpan.Kpan{
			Token: kpan.Token{
				ConsumerKey:    "<YOUR CONSUMER KEY>",
				ConsumerSecret: "<YOUR CONSUMER SECRET>",
			},
		}

		kp.Request("")

* �ڶ���: ���ݵ�һ���õ��� Kpan.Key ���ʣ�

		https://www.kuaipan.cn/api.php?ac=open&op=authorise&oauth_token=<YOUR-KEY>

  �����û���Ȩ��

* ������: ������֤��õ� KEY������ `Kpan.Token` �С�

		kp.Access(verifier)

## ʹ��

��Ȩ��Ϳ���ʹ��API�ˡ�

**����Ҫ��ʼ�� Kpan��**

	var Kpan = &kpan.Kpan{
		Token: kpan.Token{
			ConsumerKey:    "<YOUR CONSUMER KEY>",
			ConsumerSecret: "<YOUR CONSUMER SECRET>",
			Key:            "<YOUR KEY>",
			Secret:         "<YOUR SECRET>",
		},
		Root:       "app_folder",
	}

**�ϴ��ļ���**

	Kpan.Upload("/123.txt", []byte{"123456"}, true)

���ڿ������������ļ� 123.txt, ����Ϊ 123456.

**�����ļ���**

	data, err := Kpan.Download("/123.txt")

���ظղ��ϴ����ļ���data ������Ϊ 123456��

