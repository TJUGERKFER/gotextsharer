package main

import (
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
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type MessageBody struct {
	Token   string `json:"token"`
	Uid     string `json:"userid"`
	Message string `json:"message"`
}

type Profile struct {
	AdminPwd string   `json:"adminpassword"`
	UserList []string `json:"userlist"`
}

//go:embed static
var static embed.FS

func readFileHandle(w http.ResponseWriter, r *http.Request, token string, outputAsHTML bool) {
	// 从请求的查询字符串中获取token参数

	// 如果token为空，返回一个400状态码和一个错误的消息
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "token参数不能为空")
		return
	}

	cwd, _ := os.Getwd()
	cwd += "/message"
	msgpath := filepath.Clean(path.Join(cwd, token))
	relpath, _ := filepath.Rel(cwd, msgpath)
	if selfpath, _ := os.Executable(); strings.Contains(relpath, "..") || msgpath == selfpath {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "非法请求")
		return
	}

	// 从以token为文件名的文件中读取内容
	data, err := os.ReadFile(msgpath)
	message := string(data)
	if err != nil {
		// 如果读取失败，返回一个500状态码和一个错误的消息
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "消息不存在或其他数据库异常")
		return
	}

	// 如果读取成功，返回一个200状态码和文件的内容
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

func main() {
	sub, _ := fs.Sub(static, "static") // 取出 static 子文件夹
	http.Handle("/", http.FileServer(http.FS(sub)))
	// 定义一个处理函数，它会在"/"路径下被调用
	profilefile, err := os.ReadFile("./profile.json") //读取用户列表
	if err != nil {
		log.Fatal(err)
	}

	var profile Profile
	json.Unmarshal(profilefile, &profile)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/write", func(w http.ResponseWriter, r *http.Request) {
		// 从请求的查询字符串中获取token和message参数

		if !(r.Method == "POST" && r.ParseForm() == nil) {
			fmt.Fprint(w, "非法请求")
			return
		}
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

		// 如果token或message为空，返回一个400状态码和一个错误的消息
		if token == "" || message == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "token和message参数不能为空")
			return
		}
		cwd, _ := os.Getwd()
		cwd += "/message"
		msgpath := filepath.Clean(path.Join(cwd, token))
		relpath, _ := filepath.Rel(cwd, msgpath)
		if selfpath, _ := os.Executable(); strings.Contains(relpath, "..") || msgpath == selfpath {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "非法请求")
			return
		}
		// 将message的内容写入到以token为文件名的文件中
		err := os.WriteFile(msgpath, []byte(message), 0644)
		if err != nil {
			// 如果写入失败，返回一个500状态码和一个错误的消息
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "写入文件失败:", err)
			return
		}
		wlog(userid + "写入" + token)
		// 如果写入成功，返回一个200状态码和一个成功的消息
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "发送成功")
	})

	// 定义一个新的处理函数，它会在"/read"路径下被调用
	http.HandleFunc("/read", func(w http.ResponseWriter, r *http.Request) {
		readFileHandle(w, r, r.URL.Query().Get("token"), false)
	})

	// 定义一个新的处理函数，它会在"/s/"路径下被调用
	http.HandleFunc("/s/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.Split(r.URL.Path, "/")
		if len(path) != 3 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		readFileHandle(w, r, path[2], true)
	})

	// 在80端口上监听并接受连接
	http.ListenAndServe(":80", nil)
}
func wlog(message string) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05") + ":" + message)
}
