package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	log "github.com/sirupsen/logrus"
	"github.com/zvlb/release-watcher/internal/providers"
)

// HTTPClient interface
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func init() {
	Client = &http.Client{}
}

var (
	// Github Api
	githubAPI_SCHEME = "https"
	githubAPI_URL    = "api.github.com/repos"
	// URI for get last release for repository
	lastReleaseReq = "releases/latest"

	// CLient
	Client HTTPClient

	getReleaseRate      = time.Hour * 3
	githubRateLimitWait = time.Hour * 1
)

type GithubProvider struct {
	Path    string `yaml:"path"`
	Release ReleaseInfo
	client  *http.Client
}

func New(path string, client *http.Client) (providers.Provider, error) {
	gp := GithubProvider{
		Path:   path,
		client: client,
	}

	if gp.client == nil {
		gp.client = &http.Client{}
	}

	// Get actual release
	var err error
	gp.Release, err = gp.getRelease()
	if err != nil {
		return nil, err
	}

	return &gp, nil
}

func (gp *GithubProvider) WatchReleases() (name, release, description, link string, err error) {
	for {
		newReleaseExist := false
		newReleaseExist, err = gp.newReleaseExist()
		if err != nil {
			return
		}

		if newReleaseExist {
			return gp.getName(), gp.Release.TagName, gp.Release.Body, gp.Release.HtmlUrl, nil
		}

		time.Sleep(getReleaseRate)
	}
}

func (gp *GithubProvider) GetName() string {
	return gp.Path
}

func (gp *GithubProvider) newReleaseExist() (bool, error) {
	newRelease, err := gp.getRelease()
	if err != nil {
		return false, err
	}

	if newRelease.TagName != gp.Release.TagName {
		gp.updateRelease(newRelease)
		return true, nil
	}
	return false, nil
}

func (gp *GithubProvider) getRelease() (ReleaseInfo, error) {
	var ri ReleaseInfo

	requestURL := fmt.Sprintf("%v://%v/%v/%v", githubAPI_SCHEME, githubAPI_URL, gp.Path, lastReleaseReq)
	log.Debugf("Start getRelease URL  %s", requestURL)
	body, err := retry.DoWithData(
		func() ([]byte, error) {
			res, err := gp.client.Get(requestURL)
			if err != nil {
				return nil, err
			}

			if res.StatusCode != http.StatusOK {
				if res.StatusCode == http.StatusForbidden {
					log.Warnf("GitHub requests for repo %v answered with bad StatusCode: %v. It's rate limit error. Wait", gp.Path, res.StatusCode)
					return nil, errRateLimit
				}
				return nil, errNo200
			}

			body, err := io.ReadAll(res.Body)
			if err != nil {
				return nil, err
			}

			return body, nil
		},
		retry.Attempts(100),
		retry.RetryIf(func(err error) bool {
			return err != errNo200
		}),
		retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
			return githubRateLimitWait
		}),
		retry.LastErrorOnly(true),
	)

	if err != nil {
		return ri, err
	}

	err = json.Unmarshal(body, &ri)
	if err != nil {
		return ri, err
	}

	return ri, nil
}

func (gp *GithubProvider) updateRelease(release ReleaseInfo) {
	gp.Release = release
}

func (gp *GithubProvider) getName() string {
	path := strings.Split(gp.Path, "/")

	return path[1]
}
