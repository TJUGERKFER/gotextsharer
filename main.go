package main

import (
	"crypto/md5"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
)

type MessageBody struct {
	Token   string `json:"token"`
	Uid     string `json:"userid"`
	Message string `json:"message"`
}

type ChangePwdStruct struct {
	NewPwd string `json:"newpassword"`
	OldPwd string `json:"oldpassword"`
}

type AddUserStruct struct {
	Uid string `json:"userid"`
	Pwd string `json:"password"`
}

type Profile struct {
	AdminPwd string   `json:"adminpassword"`
	UserList []string `json:"userlist"`
}

type MessageIndex struct {
	Userid string `json:"userid"`
	Time   string `json:"time"`
}

//go:embed static
var static embed.FS

func readFileHandle(w http.ResponseWriter, r *http.Request, token string, outputAsHTML bool) {
	// 从请求的查询字符串中获取token参数

	// 如果token为空，返回一个400状态码
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "token参数不能为空")
		return
	}
	cwd, _ := os.Getwd()
	msgpath := path.Join(cwd, "/message", fmt.Sprintf("%x", xxhash.Sum64String(token)))

	// 从以token为文件名的文件中读取内容
	data, err := os.ReadFile(msgpath)
	message := string(data)
	if err != nil {
		// 如果读取失败，返回500状态码
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "消息不存在或其他异常")
		return
	}

	// 如果读取成功，返回200状态码
	w.WriteHeader(http.StatusOK)
	if outputAsHTML {
		tmpl, _ := template.ParseFiles("./read.html")
		if err := tmpl.Execute(w, message); err != nil {
			fmt.Println(err)
		}
	} else {
		fmt.Fprint(w, message)
	}
}

// 日志函数
func wlog(message string) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05") + ":" + message)
}

func MD5(str string) string {
	data := []byte(str) //切片
	has := md5.Sum(data)
	md5str := fmt.Sprintf("%x", has) //将 []byte转成16进制
	return md5str
}

// 检查POST请求是否正确和是否已经安装程序
func requestCheck(w http.ResponseWriter, r *http.Request, ispost bool) bool {

	if !(r.Method == "POST" && r.ParseForm() == nil) {
		if ispost {
			fmt.Fprint(w, "非法请求")
			return false
		}
	}

	_, err := os.Stat("./config/install.lock")
	if err != nil {
		fmt.Fprint(w, "请先访问/setup进行安装")
		return false
	}

	return true
}

func writeIndex(messageindex map[string]MessageIndex) {
	filebuf, _ := json.Marshal(messageindex)
	err := os.WriteFile("./message/index.json", filebuf, 0666)
	if err != nil {
		log.Fatal("写入索引失败！请检查程序读写权限")
	}
}

func isFileExist(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}

func main() {
	sub, _ := fs.Sub(static, "static") // 取出 static 子文件夹
	http.Handle("/", http.FileServer(http.FS(sub)))

	wlog("程序启动，版本v1.1")

	if !isFileExist("config") {
		wlog("config目录不存在，创建中")
		os.Mkdir("config", 0666)
	}

	if !isFileExist("message") {
		wlog("message目录不存在，创建中")
		os.Mkdir("message", 0666)
	}

	profilefile, err := os.ReadFile("./config/profile.json") //读取用户列表和密码
	var profile Profile
	if err != nil {
		wlog("读取配置文件失败，可能未正确安装")
	} else {
		json.Unmarshal(profilefile, &profile)
		if err != nil {
			wlog("解析配置文件失败，可能未正确安装")
		}
	}

	messageindex := make(map[string]MessageIndex)
	messageindexfile, err := os.ReadFile("./message/index.json")
	if err != nil {
		wlog("读取消息索引文件失败，未读取而创建新索引到内存")
		var tmp MessageIndex
		tmp.Time = fmt.Sprintf("%d", time.Now().Unix())
		tmp.Userid = "exampleid"
		messageindex["examplemessage"] = tmp
	} else {
		json.Unmarshal(messageindexfile, &messageindex)
		if err != nil {
			log.Fatal("解析消息索引失败，请检查或删除文件并再次启动！")
		}

	}

	// 写入消息函数
	http.HandleFunc("/write", func(w http.ResponseWriter, r *http.Request) {

		if !requestCheck(w, r, true) {
			return
		}
		// 从POST请求读取数据
		var messagebody MessageBody
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &messagebody)
		userid := messagebody.Uid
		token := messagebody.Token
		message := messagebody.Message

		if !slices.Contains(profile.UserList, userid) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "用户不存在，请联系管理员")
			return
		}

		// 如果token为空，返回400状态
		if token == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "token参数不能为空")
			return
		}

		//获取消息路径
		cwd, _ := os.Getwd()
		hashedtoken := xxhash.Sum64String(token)
		msgpath := path.Join(cwd, "/message", fmt.Sprintf("%x", hashedtoken))
		// 判断消息token
		if _, ok := messageindex[token]; ok {
			if message != "" && messageindex[token].Userid == userid {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "该token已被你使用过，如需再次使用请先输入空内容删除消息！")
				return
			}

			if isFileExist(msgpath) && messageindex[token].Userid != userid {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, "消息token已被其他用户使用或hash碰撞,请尝试其他token")
				return
			}

			delete(messageindex, token)
			err := os.Remove(msgpath)
			if err != nil {
				fmt.Println(err)
			}
			writeIndex(messageindex)
			w.WriteHeader(http.StatusOK)
			wlog(userid + "删除" + token)
			fmt.Fprint(w, "清除消息成功！")
			return
		}

		// 添加消息到索引中
		var newindex MessageIndex

		newindex.Time = fmt.Sprintf("%d", time.Now().Unix())
		newindex.Userid = userid
		messageindex[token] = newindex

		// 保存索引
		writeIndex(messageindex)

		// 将message的内容写入到以token为文件名的文件中
		err = os.WriteFile(msgpath, []byte(message), 0644)
		if err != nil {
			// 如果写入失败，返回500状态
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "写入文件失败:", err)
			return
		}
		wlog(userid + "写入" + token)
		// 如果写入成功，返回200状态码
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "发送成功")
	})

	// 读取消息函数
	http.HandleFunc("/read", func(w http.ResponseWriter, r *http.Request) {
		readFileHandle(w, r, r.URL.Query().Get("token"), false)
	})
	http.HandleFunc("/getindex", func(w http.ResponseWriter, r *http.Request) {
		if !requestCheck(w, r, false) {
			return
		}
		if profile.AdminPwd != MD5(r.URL.Query().Get("password")) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "密码错误")
			return
		}
		indexjsonbyte, _ := json.Marshal(messageindex)
		fmt.Fprint(w, string(indexjsonbyte))
	})
	// 伪静态 重定向到/read
	http.HandleFunc("/s/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.Split(r.URL.Path, "/")
		if len(path) != 3 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		readFileHandle(w, r, path[2], true)
	})
	// 修改密码函数
	http.HandleFunc("/changepwd", func(w http.ResponseWriter, r *http.Request) {

		if !requestCheck(w, r, true) {
			return
		}

		var changepwdstruct ChangePwdStruct
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &changepwdstruct)
		// 验证原密码
		if profile.AdminPwd != MD5(changepwdstruct.OldPwd) {
			fmt.Fprint(w, "密码错误")
			return
		}
		// 写入配置文件
		profile.AdminPwd = MD5(changepwdstruct.NewPwd)
		filebuf, _ := json.Marshal(profile)
		os.WriteFile("./config/profile.json", filebuf, 0666)
		fmt.Fprint(w, "修改成功！")
		wlog("修改密码，MD5值为" + profile.AdminPwd)

	})
	// 添加用户函数
	http.HandleFunc("/adduser", func(w http.ResponseWriter, r *http.Request) {
		if !requestCheck(w, r, true) {
			return
		}

		var adduserstruct AddUserStruct
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &adduserstruct)

		if profile.AdminPwd != MD5(adduserstruct.Pwd) {
			fmt.Fprint(w, "密码错误")
			return
		}
		// 写入配置文件
		profile.UserList = append(profile.UserList, adduserstruct.Uid)
		filebuf, _ := json.Marshal(profile)
		os.WriteFile("./config/profile.json", filebuf, 0666)
		fmt.Fprint(w, "创建成功！")
		wlog("增加用户" + adduserstruct.Uid)
	})
	// 安装函数
	http.HandleFunc("/setupapi", func(w http.ResponseWriter, r *http.Request) {
		// 检测是否已经安装
		_, err := os.Stat("./config/install.lock")
		if err == nil {
			fmt.Fprint(w, "请勿重复安装！")
			return
		}

		if !(r.Method == "POST" && r.ParseForm() == nil) {
			fmt.Fprint(w, "非法请求")
			return
		}
		// 读取数据
		data, _ := io.ReadAll(r.Body)
		var newprofile Profile
		json.Unmarshal(data, &newprofile)

		if newprofile.AdminPwd == "" || len(newprofile.UserList) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "密码或userid参数不能为空")
			return
		}
		newprofile.AdminPwd = MD5(newprofile.AdminPwd)
		filebuf, _ := json.Marshal(newprofile)
		// 创建install.lock
		os.WriteFile("./config/profile.json", filebuf, 0666)
		lock, err := os.Create("./config/install.lock")
		if err != nil {
			fmt.Fprint(w, "安装锁文件创建失败！")
			return
		}
		defer lock.Close()
		profile = newprofile
		fmt.Fprint(w, "安装成功")
		wlog("系统安装成功")
	})
	// 在80端口上监听并接受连接
	http.ListenAndServe(":80", nil)
}
