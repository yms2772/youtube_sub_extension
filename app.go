package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/maxence-charriere/go-app/v7/pkg/app"
)

const (
	Server = "mokky:mokky04120@@tcp(127.0.0.1)/jamak_history"
	Port   = 8080
)

type Subdomains map[string]http.Handler

type ErrorJSON struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
}

type YoutubeJSON struct {
	URL  string `json:"url"`
	Code int    `json:"code"`
}

type SubtitleJSON struct {
	Subtitle string `json:"subtitle"`
	Code     int    `json:"code"`
}

type SaveJSON struct {
	Version string `json:"version"`
	Code    int    `json:"code"`
}

func (subdomains Subdomains) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	domainParts := strings.Split(r.Host, ".")

	if mux := subdomains[domainParts[0]]; mux != nil {
		mux.ServeHTTP(w, r)
	} else {
		http.Error(w, "Not found", 404)
	}
}

func Error(w http.ResponseWriter, err error, code int) {
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(ErrorJSON{
		Msg:  err.Error(),
		Code: code,
	})
}

func AddSubtitle(id, ip, lang string) string {
	database, err := sql.Open("mysql", Server)
	if err != nil {
		fmt.Println(err)
	}

	_, err = database.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (version int(10) NOT NULL AUTO_INCREMENT PRIMARY KEY, ip varchar(15) NOT NULL, date DATETIME NOT NULL, lang varchar(15) NOT NULL) DEFAULT CHARACTER SET utf8 COLLATE utf8_general_ci`, id))
	if err != nil {
		fmt.Println(err)
	}

	_, err = database.Exec(fmt.Sprintf(`INSERT INTO %s (ip, date, lang) VALUE ('%s', '%s', '%s')`, id, ip, time.Now().Format("2006-01-02 15:04:05"), lang))
	if err != nil {
		fmt.Println(err)
	}

	database.Close()

	return ""
}

func GetLastVersion(id string) int {
	database, _ := sql.Open("mysql", Server)
	defer database.Close()

	var version int
	err := database.QueryRow(fmt.Sprintf(`SELECT version FROM %s ORDER BY version DESC LIMIT 1;`, id)).Scan(&version)
	if err != nil {
		fmt.Println(err)
	}

	return version
}

func API(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		Error(w, err, 999)
	}

	if r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	call := r.FormValue("call")

	switch call {
	case "youtube": // 100
		id := r.FormValue("id")

		if len(id) == 0 {
			Error(w, fmt.Errorf(""), 100)
			return
		}

		regx, _ := regexp.Compile(`"itag":18,"url":"(.*?)"`)

		fmt.Printf("다운로드: https://www.youtube.com/get_video_info?video_id=%s&eurl=https://youtube.googleapis.com/v/%s\n", id, id)

		client := http.Client{}

		req, err := http.NewRequest("GET", fmt.Sprintf("https://www.youtube.com/get_video_info?video_id=%s&eurl=https://youtube.googleapis.com/v/%s", id, id), nil)
		if err != nil {
			Error(w, err, 101)
			return
		}

		req.Header.Add("Content-Type", "text/html; charset=utf-8")
		req.Header.Add("Access-Control-Allow-Origin", "*")
		req.Header.Add("Access-Control-Allow-Headers", "Content-Type, Origin, Accept, token")
		req.Header.Add("Access-Control-Allow-Methods", "GET, OPTIONS")

		resp, err := client.Do(req)
		if err != nil || resp.StatusCode == 404 {
			Error(w, err, 102)
			return
		}

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		decodedBody, _ := url.QueryUnescape(string(body))
		ytURLs := regx.FindStringSubmatch(decodedBody)

		if len(ytURLs) < 2 {
			Error(w, err, 103)
			return
		}

		err = json.NewEncoder(w).Encode(YoutubeJSON{
			URL:  strings.ReplaceAll(ytURLs[1], "\\u0026", "&"),
			Code: 0,
		})
		if err != nil {
			Error(w, err, 199)
			return
		}
	case "subtitle": // 200
		platform := r.FormValue("platform")
		id := r.FormValue("id")
		lang := r.FormValue("lang")

		if len(platform) == 0 || len(id) == 0 || len(lang) == 0 {
			Error(w, fmt.Errorf(""), 200)
			return
		}

		file, err := ioutil.ReadFile(fmt.Sprintf("/home/ubuntu/jamak/subtitle/%s/%s/%s.srt", platform, id, lang))
		if err != nil {
			Error(w, err, 201)
			return
		}

		err = json.NewEncoder(w).Encode(SubtitleJSON{
			Subtitle: string(file),
			Code:     0,
		})
		if err != nil {
			Error(w, err, 299)
			return
		}
	case "save": // 300
		ip := r.FormValue("ip")
		platform := r.FormValue("platform")
		id := r.FormValue("id")
		lang := r.FormValue("lang")
		subtitle := r.FormValue("subtitle")

		if len(ip) == 0 || len(platform) == 0 || len(id) == 0 || len(lang) == 0 || len(subtitle) == 0 {
			Error(w, fmt.Errorf(""), 300)
			return
		}

		if err := os.MkdirAll(fmt.Sprintf("/home/ubuntu/jamak/subtitle/%s/%s/version", platform, id), 0777); err != nil {
			fmt.Println(err)

			Error(w, fmt.Errorf(""), 301)
			return
		}

		AddSubtitle(id, ip, lang)
		version := GetLastVersion(id)

		orgFile := fmt.Sprintf("/home/ubuntu/jamak/subtitle/%s/%s/version/r%d-%s.srt", platform, id, version, lang)
		file := fmt.Sprintf("/home/ubuntu/jamak/subtitle/%s/%s/%s.srt", platform, id, lang)

		input, _ := ioutil.ReadFile(file)

		if err := ioutil.WriteFile(orgFile, input, 0777); err != nil {
			Error(w, err, 304)
			return
		}

		err = ioutil.WriteFile(file, []byte(subtitle), 0777)
		if err != nil {
			Error(w, err, 305)
			return
		}

		err = json.NewEncoder(w).Encode(SaveJSON{
			Version: fmt.Sprintf("r%d", version),
			Code:    0,
		})
		if err != nil {
			Error(w, err, 399)
			return
		}
	default:
		Error(w, fmt.Errorf("%s", "Not Found"), 1)
	}
}

func main() {
	h := &app.Handler{
		Title: "자막 편집기",
		Styles: []string{
			"/web/app.css",
		},
	}

	subdomains := make(Subdomains)
	subdomains["editor"] = h

	mux := http.NewServeMux()
	mux.Handle("/", subdomains)
	mux.HandleFunc("/api", API)

	log.Printf("Running server on %d port!", Port)

	if err := http.ListenAndServe(fmt.Sprintf("localhost:%d", Port), mux); err != nil {
		panic(err)
	}
}
