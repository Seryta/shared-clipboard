package main

import (
	"errors"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	log "github.com/sirupsen/logrus"
)

const (
	letterBytes            = "abcdefghijklmnopqrstuvwxyz"
	clipboardParentDirPath = "/shared-clipboard"
	clipboardDirName       = "/shared-clipboard/clipboard-files"

	envClipNumberLimit = "CLIPFILE_NUMBER_LIMIT"
	envClipTimeLimit   = "CLIPFILE_TIME_LIMIT"
)

var (
	clipFileNumber      = 0
	clipFileNumberLimit = 1000
	clipFileTimeLimit   = 15
)

func main() {
	if os.Getenv(envClipNumberLimit) != "" {
		if tmp, err := strconv.Atoi(os.Getenv(envClipNumberLimit)); err != nil {
			log.Errorf("环境变量 %v 获取错误，报错：%v ，提示：此环境变量用于限制临时剪贴板的数量，应是数值，默认 1000", envClipNumberLimit, err)
		} else {
			clipFileNumberLimit = tmp
		}
	}
	log.Infof("剪贴板数量限制：%v", clipFileNumberLimit)
	if os.Getenv(envClipTimeLimit) != "" {
		if tmp, err := strconv.Atoi(os.Getenv(envClipTimeLimit)); err != nil {
			log.Errorf("环境变量 %v 获取错误，报错：%v ，提示：此环境变量用于限制每个临时剪贴板存在的时间，应是数值，默认 15 (单位：分钟)", envClipTimeLimit, err)
		} else {
			clipFileTimeLimit = tmp
		}
	}
	log.Infof("每个剪贴板存在时间限制：%v 分钟", clipFileTimeLimit)

	initDir()
	go cronCleanFiles()

	sPipe := alice.New()
	router := httprouter.New()

	router.Handler("GET", "/", sPipe.ThenFunc(index))
	router.ServeFiles("/ui/*filepath", http.Dir("./web"))
	router.Handler("GET", "/api/:id", sPipe.ThenFunc(getMsg))
	router.Handler("POST", "/api/:id", sPipe.ThenFunc(postMsg))

	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Error(err)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "ui", http.StatusSeeOther)
}

func initDir() {
	os.Mkdir(clipboardParentDirPath, 0666)
	if _, err := os.Stat(clipboardDirName); os.IsNotExist(err) {
		log.Println("创建共享剪贴板文件目录")
		if err := os.Mkdir(clipboardDirName, 0700); err != nil {
			log.Fatalln("创建共享剪贴板目录失败，错误：", err)
		}
	} else if err == nil {
		log.Println("清理共享剪贴板文件目录文件")
		cleanFiles()
	} else {
		log.Fatalln("获取共享剪贴板文件目录失败，错误：", err)
	}
	log.Println("初始化清理文件完成")
}

func cronCleanFiles() {
	ticker := time.NewTicker(time.Minute * 1)
	for range ticker.C {
		cleanFiles()
	}
}

func cleanFiles() {
	entries, err := os.ReadDir(clipboardDirName)
	if err != nil {
		log.Errorf("获取共享剪贴板文件列表失败，错误：%v", err)
		return
	}
	nowTime := time.Now()
	clipFileNumber = len(entries)
	for _, entry := range entries {
		if !entry.IsDir() {
			fileInfo, err := os.Stat(filepath.Join(clipboardDirName, entry.Name()))
			if err != nil {
				log.Errorf("获取共享剪贴板文件信息失败，文件：%v, 错误：%v", entry.Name(), err)
				continue
			}
			stat_t := fileInfo.Sys().(*syscall.Stat_t)
			atime := time.Unix(stat_t.Atim.Sec, 0)
			ctime := time.Unix(stat_t.Ctim.Sec, 0)
			if nowTime.Sub(atime) > time.Minute*time.Duration(clipFileTimeLimit) && nowTime.Sub(ctime) > time.Minute*time.Duration(clipFileTimeLimit) {
				if err := os.Remove(filepath.Join(clipboardDirName, entry.Name())); err != nil {
					log.Errorf("删除超时共享剪贴板文件失败，文件：%v, 错误：%v", entry.Name(), err)
				}
			}
		}
	}
}

func genID() (string, error) {
	for i := 0; i < 10; i++ {
		id := randStringBytes(5)
		if _, err := os.Stat(filepath.Join(clipboardDirName, id)); !os.IsNotExist(err) {
			log.Infof("查看共享剪贴板文件失败, ID: %s, 错误: %v", id, err)
			continue
		}
		if _, err := os.Create(filepath.Join(clipboardDirName, id)); err != nil {
			log.Infof("创建共享剪贴板文件失败, ID: %s, 错误: %v", id, err)
			continue
		}
		return id, nil
	}
	return "", errors.New("当前服务器繁忙，请稍后再试")
}

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func getMsg(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	id := params.ByName("id")
	f, err := os.ReadFile(filepath.Join(clipboardDirName, id))
	if err == nil {
		os.Chtimes(filepath.Join(clipboardDirName, id), time.Now(), time.Now())
		w.WriteHeader(http.StatusOK)
		w.Write(f)
		return
	}
	if os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("ID不存在，请确认请求信息是否正确"))
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte(err.Error()))
}

func postMsg(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	id := params.ByName("id")
	if id == "new" {
		if clipFileNumber > clipFileNumberLimit {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("当前服务器繁忙，请稍后再试"))
			return
		}
		var err error
		if id, err = genID(); err != nil {
			log.Errorf("生成ID失败，错误：%v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("生成ID失败"))
			return
		}
	}
	f, err := os.OpenFile(filepath.Join(clipboardDirName, id), os.O_WRONLY|os.O_TRUNC, 0600)
	defer f.Close()
	if err == nil {
		body, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			log.Errorf("读取请求体失败，ID：%v, 错误：%v", id, err)
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("读取请求体失败"))
			return
		}

		if _, err := f.Write(body); err != nil {
			log.Errorf("写入共享剪贴板信息失败，ID: %v, 错误：%v", id, err)
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("写入共享剪贴板信息失败"))
			return
		}
		w.WriteHeader(http.StatusOK)
		if params.ByName("id") == "new" {
			w.Write([]byte(id))
		} else {
			w.Write([]byte("信息共享成功"))
		}
	} else if os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("请求的剪贴内容已不存在"))
	} else {
		log.Errorf("打开共享剪贴板文件失败，ID：%v, 错误：%v", id, err)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("打开共享内容失败"))
	}
}
