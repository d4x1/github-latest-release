package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	homePage  = "https://github-latest-release.vercel.app"
	githubAPI = "https://api.github.com/repos/%s/releases"
)

type GitHubReleasesResp struct {
	Url       string `json:"url"`
	AssetsUrl string `json:"assets_url"`
	UploadUrl string `json:"upload_url"`
	HtmlUrl   string `json:"html_url"`
	Id        int    `json:"id"`
	Author    struct {
		Login             string `json:"login"`
		Id                int    `json:"id"`
		NodeId            string `json:"node_id"`
		AvatarUrl         string `json:"avatar_url"`
		GravatarId        string `json:"gravatar_id"`
		Url               string `json:"url"`
		HtmlUrl           string `json:"html_url"`
		FollowersUrl      string `json:"followers_url"`
		FollowingUrl      string `json:"following_url"`
		GistsUrl          string `json:"gists_url"`
		StarredUrl        string `json:"starred_url"`
		SubscriptionsUrl  string `json:"subscriptions_url"`
		OrganizationsUrl  string `json:"organizations_url"`
		ReposUrl          string `json:"repos_url"`
		EventsUrl         string `json:"events_url"`
		ReceivedEventsUrl string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"author"`
	NodeId          string    `json:"node_id"`
	TagName         string    `json:"tag_name"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	Draft           bool      `json:"draft"`
	Prerelease      bool      `json:"prerelease"`
	CreatedAt       time.Time `json:"created_at"`
	PublishedAt     string    `json:"published_at"`
	Assets          []struct {
		Url      string      `json:"url"`
		Id       int         `json:"id"`
		NodeId   string      `json:"node_id"`
		Name     string      `json:"name"`
		Label    interface{} `json:"label"`
		Uploader struct {
			Login             string `json:"login"`
			Id                int    `json:"id"`
			NodeId            string `json:"node_id"`
			AvatarUrl         string `json:"avatar_url"`
			GravatarId        string `json:"gravatar_id"`
			Url               string `json:"url"`
			HtmlUrl           string `json:"html_url"`
			FollowersUrl      string `json:"followers_url"`
			FollowingUrl      string `json:"following_url"`
			GistsUrl          string `json:"gists_url"`
			StarredUrl        string `json:"starred_url"`
			SubscriptionsUrl  string `json:"subscriptions_url"`
			OrganizationsUrl  string `json:"organizations_url"`
			ReposUrl          string `json:"repos_url"`
			EventsUrl         string `json:"events_url"`
			ReceivedEventsUrl string `json:"received_events_url"`
			Type              string `json:"type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"uploader"`
		ContentType        string    `json:"content_type"`
		State              string    `json:"state"`
		Size               int       `json:"size"`
		DownloadCount      int       `json:"download_count"`
		CreatedAt          time.Time `json:"created_at"`
		UpdatedAt          time.Time `json:"updated_at"`
		BrowserDownloadUrl string    `json:"browser_download_url"`
	} `json:"assets"`
	TarballUrl string `json:"tarball_url"`
	ZipballUrl string `json:"zipball_url"`
	Body       string `json:"body"`
}

func (r *GitHubReleasesResp) AssertByName(name string) (string, error) {
	if len(name) == 0 {
		return "", errors.New("release filename is empty")
	}
	if r == nil {
		return "", errors.New("github api response is empty")
	}
	if len(r.Assets) == 0 {
		return "", errors.New("asset list is empty")
	}
	for _, a := range r.Assets {
		if a.Name == name {
			return a.BrowserDownloadUrl, nil
		}
	}
	return "", errors.New("not found")
}

func GetLatestRelease(resp []*GitHubReleasesResp) *GitHubReleasesResp {
	if len(resp) == 0 {
		return nil
	}
	if len(resp) == 1 {
		return resp[0]
	}
	max := resp[0]
	for _, r := range resp {
		if TimeStrToUnix(r.PublishedAt) > TimeStrToUnix(max.PublishedAt) {
			max = r
		}
	}
	return max
}

func TimeStrToUnix(s string) int64 {
	if s == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		log.Printf("time parse: %s, err: %s", s, err)
	}
	return t.Unix()
}

func NewResp(code int, msg string) map[string]interface{} {
	resp := make(map[string]interface{})
	resp["code"] = code
	resp["msg"] = msg
	return resp
}

func WriteJson(w http.ResponseWriter, data interface{}) {
	b, _ := json.Marshal(data)
	w.Write(b)
}

func DownloadLatestGithubRelease(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		repoName := r.URL.Query().Get("repo")
		if repoName == "" {
			// 需要指定repo才能用，引导到首页
			WriteJson(w, NewResp(-1, fmt.Sprintf("please provide repo name, for more detail, visit: %s", homePage)))
			return
		}
		if len(strings.Split(repoName, "/")) != 2 {
			WriteJson(w, NewResp(-1, fmt.Sprintf("please check your repo name(%s), for more detail, visit: %s", repoName, homePage)))
			return
		}
		// 请求实际的 API
		api := fmt.Sprintf(githubAPI, repoName)
		log.Printf("repo name: %s, api: %s", repoName, api)
		client := http.DefaultClient
		req, err := http.NewRequest(http.MethodGet, api, nil)
		if err != nil {
			log.Printf("new http request, api: %s, err: %+v", api, err)
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("client do http request, req: %+v, err: %+v", req, err)
			return
		}
		defer resp.Body.Close()
		var respStruct []*GitHubReleasesResp
		bodyData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("ioutil read resp body, resp: %+v, err: %+v", resp, err)
			return
		}
		if err := json.Unmarshal(bodyData, &respStruct); err != nil {
			log.Printf("json unmarshal resp data, resp: %s, err: %+v", bodyData, err)
			return
		}

		ret := GetLatestRelease(respStruct)
		if ret == nil {
			WriteJson(w, NewResp(-1, fmt.Sprintf("repo: %s has no release jet", repoName)))
			return
		}
		downloadURL, err := ret.AssertByName(r.URL.Query().Get("name"))
		if err != nil {
			WriteJson(w, NewResp(-1, fmt.Sprintf("get repo: %s's asset err: %s", repoName, err)))
			return
		}
		log.Printf("download link: %s", downloadURL)
		http.Redirect(w, r, downloadURL, http.StatusTemporaryRedirect)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Header().Set("Allow", http.MethodGet)
	}
}
