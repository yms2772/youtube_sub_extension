package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/asticode/go-astisub"
	"github.com/maxence-charriere/go-app/v7/pkg/app"
)

const (
	ApiServer = "https://editor.jamak.icu/api"
)

var (
	subTextClicked bool
)

type ResultJSON struct {
	Code     int    `json:"code"`
	Subtitle string `json:"subtitle"`
	URL      string `json:"url"`
	Version  string `json:"version"`
	Msg      string `json:"msg"`
}

type player struct {
	app.Compo

	subtitle
	editor
	control
	user

	video app.Value

	videoLoaded bool

	youtubeID     string
	youtubeURL    string
	youtubeStart  float64
	youtubeWidth  int
	youtubeHeight int
}

type subtitle struct {
	app.Compo

	content app.RangeLoop

	youtubeSrtSub []*srtSub

	youtubeSubtitle           string
	youtubeSubtitleEndAt      float64
	youtubeSubtitleMarginTop  int
	youtubeSubtitleMarginLeft int
	youtubeSubtitleWidth      int

	hidden bool
}

type editor struct {
	app.Compo

	startAtError string
	endAtError   string
	subTextError string
	height       int
}

type control struct {
	app.Compo

	marginLeft int

	playPause   string
	currentTime string
	totalTime   string
}

type user struct {
	app.Compo

	ip string
}

type srtSub struct {
	Index   int
	Text    string
	StartAt time.Duration
	EndAt   time.Duration
}

func IsSubExist(platform, id, lang string) (string, int) {
	data := url.Values{}
	data.Add("call", "subtitle")
	data.Add("platform", platform)
	data.Add("id", id)
	data.Add("lang", lang)

	resp, err := http.PostForm(ApiServer, data)
	if err != nil || resp.StatusCode != 200 {
		fmt.Println(err)
		return "", 0
	}

	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	var resultJSON ResultJSON
	err = json.Unmarshal(body, &resultJSON)
	if err != nil {
		fmt.Println(err)
		return "", 0
	}

	return resultJSON.Subtitle, resp.StatusCode
}

func (p *player) DelSub(i int) {
	p.subtitle.youtubeSrtSub = append(p.subtitle.youtubeSrtSub[:i], p.subtitle.youtubeSrtSub[i+1:]...)
}

func (p *player) AddSub(i int, sub *srtSub) {
	p.subtitle.youtubeSrtSub = append(p.subtitle.youtubeSrtSub[:i+1], p.subtitle.youtubeSrtSub[i:]...)
	p.subtitle.youtubeSrtSub[i+1] = sub
}

func (p *player) Play() {
	p.video.Call("play")
	p.control.playPause = "play-play"
}

func (p *player) Pause() {
	p.video.Call("pause")
	p.control.playPause = "play-pause"
}

func (p *player) LoadSubList() app.RangeLoop {
	return app.Range(p.subtitle.youtubeSrtSub).Slice(func(i int) app.UI {
		startAt := p.subtitle.youtubeSrtSub[i].StartAt.Milliseconds()
		endAt := p.subtitle.youtubeSrtSub[i].EndAt.Milliseconds()

		// 시작시간
		startAth := startAt / 3600000
		startAt = startAt - (3600000 * startAth)
		startAtm := startAt / 60000
		startAt = startAt - (60000 * startAtm)
		startAts := startAt / 1000
		startAt = startAt - (1000 * startAts)

		// 종료시간
		endAth := endAt / 3600000
		endAt = endAt - (3600000 * endAth)
		endAtm := endAt / 60000
		endAt = endAt - (60000 * endAtm)
		endAts := endAt / 1000
		endAt = endAt - (1000 * endAts)

		return app.Div().Body( // editor-container
			app.Div().Body( // 자막 에디터 Div
				app.Div().Body( // Time Input
					app.Input().
						Class("sub-timeline").
						Value(fmt.Sprintf("%02d:%02d:%02d.%03d", startAth, startAtm, startAts, startAt)).
						OnInput(func(ctx app.Context, e app.Event) {
							setTime, err := time.Parse("15:04:05.000", ctx.JSSrc.JSValue().Get("value").String())
							if err != nil {
								return
							}

							timeDuration := ((float64(setTime.Hour()*3600) + float64(setTime.Minute()*60) + float64(setTime.Second())) + (float64(setTime.Nanosecond()) / 1000000000)) * 1000000000

							p.subtitle.youtubeSrtSub[i].StartAt = time.Duration(timeDuration)
							p.Update()

							fmt.Println("시간 설정: " + p.subtitle.youtubeSrtSub[i].StartAt.String())
						}),
					app.Input().
						Class("sub-timeline").
						Value(fmt.Sprintf("%02d:%02d:%02d.%03d", endAth, endAtm, endAts, endAt)).
						OnInput(func(ctx app.Context, e app.Event) {
							setTime, err := time.Parse("15:04:05.000", ctx.JSSrc.JSValue().Get("value").String())
							if err != nil {
								return
							}

							timeDuration := ((float64(setTime.Hour()*3600) + float64(setTime.Minute()*60) + float64(setTime.Second())) + (float64(setTime.Nanosecond()) / 1000000000)) * 1000000000

							p.subtitle.youtubeSrtSub[i].EndAt = time.Duration(timeDuration)
							p.Update()

							fmt.Println("시간 설정: " + p.subtitle.youtubeSrtSub[i].EndAt.String())
						}),
				).
					Class("sub-time"),
				app.Div().Body( // Textarea
					app.Textarea().
						Name("sub").
						Class("form-control").
						Text(p.subtitle.youtubeSrtSub[i].Text).
						Placeholder("자막을 입력해주세요").
						OnClick(func(ctx app.Context, e app.Event) {
							fmt.Printf("[%s~%s] 자막 클릭: %s\n", p.subtitle.youtubeSrtSub[i].StartAt.String(), p.subtitle.youtubeSrtSub[i].EndAt.String(), p.subtitle.youtubeSrtSub[i].Text)
							subTextClicked = true
							p.subtitle.youtubeSubtitle = p.subtitle.youtubeSrtSub[i].Text
							p.youtubeStart = p.subtitle.youtubeSrtSub[i].StartAt.Seconds()
							p.Update()
						}).
						OnInput(func(ctx app.Context, e app.Event) {
							p.Pause()
							p.subtitle.youtubeSrtSub[i].Text = ctx.JSSrc.JSValue().Get("value").String()
							p.Update()
						}),
				).
					Class("sub-text"),
				app.Div().Body(
					app.A().
						Class("sub-del").
						Href("#").
						OnClick(func(ctx app.Context, e app.Event) {
							fmt.Printf("%d번 자막 삭제\n", i+1)
							p.DelSub(i)
							p.subtitle.content = p.LoadSubList()
							p.Update()
						}),
					app.A().
						Class("sub-add").
						Href("#").
						OnClick(func(ctx app.Context, e app.Event) {
							fmt.Printf("%d번 자막 추가\n", i+1)
							p.AddSub(i, &srtSub{
								Index:   i + 1,
								Text:    "",
								StartAt: time.Duration(p.video.Get("currentTime").Float() * 1000000000),
								EndAt:   time.Duration(p.video.Get("currentTime").Float() * 1000000000),
							})
							p.subtitle.content = p.LoadSubList()
							p.Update()
						}),
				).
					Class("sub-buttons"),
			).
				Class("sub-card"),
		).
			Class("editor-container")
	})
}

func (p *player) OnMount(app.Context) {
	fmt.Println("구성요소 mount")
}

func (p *player) OnDismount() {
	fmt.Println("구성요소 dismount")
}

func (p *player) OnNav(_ app.Context, u *url.URL) {
	p.youtubeID = u.Query().Get("id")

	fmt.Println("ID: " + p.youtubeID)

	if len(p.youtubeID) != 0 {
		fmt.Println("유튜브 영상 정보 가져오는 중...")
		data := url.Values{}
		data.Add("call", "youtube")
		data.Add("id", p.youtubeID)

		resp, err := http.PostForm(ApiServer, data)
		if err != nil {
			fmt.Println("API 불러오기 실패 (resp)")
			fmt.Println(err)

			return
		}

		ytBody, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		var resultJSON ResultJSON
		err = json.Unmarshal(ytBody, &resultJSON)
		if err != nil {
			fmt.Println("실패")

			return
		}

		ytURL := resultJSON.URL

		if len(ytURL) == 0 {
			fmt.Println("실패")

			return
		}

		body, statusCode := IsSubExist("youtube", p.youtubeID, "ko")

		if statusCode == 200 {
			fmt.Println("자막 발견")
			srt, err := astisub.ReadFromSRT(strings.NewReader(body))
			if err != nil {
				fmt.Println(err)

				return
			}

			for _, item := range srt.Items {
				p.subtitle.youtubeSrtSub = append(p.subtitle.youtubeSrtSub, &srtSub{
					Index:   item.Index,
					Text:    item.String(),
					StartAt: item.StartAt,
					EndAt:   item.EndAt,
				})
			}
		} else {
			p.subtitle.youtubeSrtSub = append(p.subtitle.youtubeSrtSub, &srtSub{
				Index:   0,
				Text:    "Text here",
				StartAt: 0,
				EndAt:   0,
			})
		}

		p.control.playPause = "play-pause"
		p.youtubeURL = ytURL
		p.content = p.LoadSubList()
	}

	fmt.Println("페이지 업데이트")
	p.Update()
}

func (p *player) Render() app.UI {
	var setSubText string

	return app.Div().Body(
		app.Div().Body( // 왼쪽
			app.Div().Body( // 유튜브 영상
				app.Video().
					Src(p.youtubeURL).
					ID("player").
					Class("player").
					Width(p.youtubeWidth).
					Height(p.youtubeHeight).
					AutoPlay(true).
					OnTimeUpdate(func(ctx app.Context, e app.Event) {
						p.video = ctx.Src.JSValue()
						currentTime := p.video.Get("currentTime").Float()
						duration := p.video.Get("duration").Float()
						p.control.currentTime = fmt.Sprintf("%02d:%02d:%02d.%03d", int(currentTime)/3600, (int(currentTime)/60)-((int(currentTime)/3600)*60), int(currentTime)%60, int((currentTime-float64(int(currentTime)))*1000))
						p.control.totalTime = fmt.Sprintf("%02d:%02d:%02d.%03d", int(duration)/3600, (int(duration)/60)-((int(duration)/3600)*60), int(duration)%60, int((duration-float64(int(duration)))*1000))

						if subTextClicked {
							fmt.Printf("시간 설정: %f초\n", p.youtubeStart)
							p.video.Set("currentTime", p.youtubeStart)
							subTextClicked = false
						}

						for _, item := range p.subtitle.youtubeSrtSub {
							if currentTime >= item.StartAt.Seconds() && currentTime < item.EndAt.Seconds() {
								if setSubText == item.Text {
									break
								}

								fmt.Println("자막 설정: " + item.Text)
								setSubText = item.Text
								p.subtitle.hidden = false
								p.subtitle.youtubeSubtitleEndAt = item.EndAt.Seconds()
								p.subtitle.youtubeSubtitle = setSubText

								break
							}
						}

						if currentTime > p.subtitle.youtubeSubtitleEndAt && p.subtitle.hidden == false {
							fmt.Println("자막 숨기기")
							p.subtitle.hidden = true
						}

						p.Update()
					}).
					OnLoadedData(func(ctx app.Context, e app.Event) {
						go func() {
							var prevWindowW, prevWindowH int

							for {
								windowW, windowH := app.Window().Size()

								if prevWindowW != windowW || prevWindowH != windowH {
									prevWindowW = windowW
									prevWindowH = windowH

									p.youtubeWidth = int(float64(windowW) * float64(9) / float64(16) * 1.12)
									p.youtubeHeight = int(float64(windowH) * float64(9) / float64(16) * 1.12)
									p.subtitle.youtubeSubtitleMarginTop = int(float64(p.youtubeHeight) * 0.95)
									p.subtitle.youtubeSubtitleMarginLeft = int(float64(p.youtubeWidth/2) * 0.3)
									p.subtitle.youtubeSubtitleWidth = int(float64(p.youtubeWidth) * 0.7)
									p.editor.height = int(float64(windowH) * 0.85)
									p.Update()

									fmt.Printf("가로: %d -> %d\n", windowW, p.youtubeWidth)
									fmt.Printf("세로: %d -> %d\n", windowH, p.youtubeHeight)
								}

								time.Sleep(100 * time.Nanosecond)
							}
						}()
					}),
			).
				Class("display-video"),
			app.Input().
				Class("display-subtitle").
				ContentEditable(false).
				ReadOnly(true).
				Hidden(p.subtitle.hidden).
				Style("margin-top", fmt.Sprintf("%dpx", p.subtitle.youtubeSubtitleMarginTop)).
				Style("margin-left", fmt.Sprintf("%dpx", p.subtitle.youtubeSubtitleMarginLeft)).
				Style("width", fmt.Sprintf("%dpx", p.subtitle.youtubeSubtitleWidth)).
				Value(p.subtitle.youtubeSubtitle),
			app.Div().Body(
				app.Span().
					Class("play-rewind").
					Title("5초 뒤로").
					OnClick(func(ctx app.Context, e app.Event) {
						fmt.Println("5초 뒤로")
						p.video.Set("currentTime", p.video.Get("currentTime").Float()-5)
					}),
				app.Span().
					Class(p.control.playPause).
					Title("재생/정지").
					OnClick(func(ctx app.Context, e app.Event) {
						if p.control.playPause == "play-play" {
							fmt.Println("정지")
							p.Pause()
						} else {
							fmt.Println("재생")
							p.Play()
						}

						p.Update()
					}),
				app.Span().
					Class("play-forward").
					Title("5초 앞으로").
					OnClick(func(ctx app.Context, e app.Event) {
						fmt.Println("5초 앞으로")
						p.video.Set("currentTime", p.video.Get("currentTime").Float()+5)
					}),
				app.Div().Body(
					app.Span().Body(
						app.Text(p.control.currentTime),
					).
						Class("play-time-current"),
					app.Span().Body(
						app.Text("/"),
					).
						Class("play-time-divider"),
					app.Span().Body(
						app.Text(p.control.totalTime),
					).
						Class("play-time-total"),
				).
					Class("play-time"),
			).
				Class("play-control"),
		).
			Class("display-left"),
		app.Div().Body( // 오른쪽
			app.Div().Body( // 자막 에디터
				p.subtitle.content,
			).
				Class("display-subtitle-editor").
				Style("height", fmt.Sprintf("%dpx", p.editor.height)),
			app.Div().Body( // 저장
				app.Script().Src("https://jsgetip.appspot.com/"),
				app.Button().Body(
					app.Text("Save"),
				).
					Class("btn btn-blue").
					Type("button").
					OnClick(func(ctx app.Context, e app.Event) {
						fmt.Println("저장")

						var youtubeSrtRaw string

						for _, item := range p.subtitle.youtubeSrtSub {
							startAt := item.StartAt.Milliseconds()
							endAt := item.EndAt.Milliseconds()

							// 시작시간
							startAth := startAt / 3600000
							startAt = startAt - (3600000 * startAth)
							startAtm := startAt / 60000
							startAt = startAt - (60000 * startAtm)
							startAts := startAt / 1000
							startAt = startAt - (1000 * startAts)

							// 종료시간
							endAth := endAt / 3600000
							endAt = endAt - (3600000 * endAth)
							endAtm := endAt / 60000
							endAt = endAt - (60000 * endAtm)
							endAts := endAt / 1000
							endAt = endAt - (1000 * endAts)

							youtubeSrtRaw += fmt.Sprintf("%d\n"+
								"%s --> %s\n"+
								"%s\n"+
								"\n",
								item.Index,
								fmt.Sprintf("%02d:%02d:%02d,%03d", startAth, startAtm, startAts, startAt),
								fmt.Sprintf("%02d:%02d:%02d,%03d", endAth, endAtm, endAts, endAt),
								item.Text,
							)
						}

						p.user.ip = app.Window().Call("ip").String()
						data := url.Values{}
						data.Add("call", "save")
						data.Add("ip", p.user.ip)
						data.Add("platform", "youtube")
						data.Add("id", p.youtubeID)
						data.Add("lang", "ko")
						data.Add("subtitle", youtubeSrtRaw)

						resp, err := http.PostForm(ApiServer, data)
						if err != nil || resp.StatusCode != 200 {
							app.Window().Call("alert", "저장 실패")

							return
						}

						body, _ := ioutil.ReadAll(resp.Body)
						resp.Body.Close()

						var resultJSON ResultJSON
						err = json.Unmarshal(body, &resultJSON)
						if err != nil || resultJSON.Code != 0 {
							app.Window().Call("alert", "저장 실패")

							return
						}

						app.Window().Call("alert", fmt.Sprintf("저장 완료\n"+
							"버전: %s",
							resultJSON.Version,
						))
					}),
			).
				Class("editor-button"),
		).
			Class("display-right"),
	).
		Class("display-main")
}

func main() {
	app.Route("/", &player{})
	app.Run()
}
