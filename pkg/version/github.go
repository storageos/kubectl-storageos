package version

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sort"
	"sync"
	"time"
)

const (
	operatorReleasesUrl        = "https://api.github.com/repos/storageos/operator/releases"
	clusterOperatorReleasesUrl = "https://api.github.com/repos/storageos/cluster-operator/releases"
)

var (
	operatorLatestVersion      string
	clusterOperatorLastVersion string

	fetchOperatorVersionOnce        = sync.Once{}
	fetchClusterOperatorVersionOnce = sync.Once{}
)

type GithubRelease struct {
	URL       string `json:"url,omitempty"`
	AssetsURL string `json:"assets_url,omitempty"`
	UploadURL string `json:"upload_url,omitempty"`
	HTMLURL   string `json:"html_url,omitempty"`
	ID        int    `json:"id,omitempty"`
	Author    struct {
		Login             string `json:"login,omitempty"`
		ID                int    `json:"id,omitempty"`
		NodeID            string `json:"node_id,omitempty"`
		AvatarURL         string `json:"avatar_url,omitempty"`
		GravatarID        string `json:"gravatar_id,omitempty"`
		URL               string `json:"url,omitempty"`
		HTMLURL           string `json:"html_url,omitempty"`
		FollowersURL      string `json:"followers_url,omitempty"`
		FollowingURL      string `json:"following_url,omitempty"`
		GistsURL          string `json:"gists_url,omitempty"`
		StarredURL        string `json:"starred_url,omitempty"`
		SubscriptionsURL  string `json:"subscriptions_url,omitempty"`
		OrganizationsURL  string `json:"organizations_url,omitempty"`
		ReposURL          string `json:"repos_url,omitempty"`
		EventsURL         string `json:"events_url,omitempty"`
		ReceivedEventsURL string `json:"received_events_url,omitempty"`
		Type              string `json:"type,omitempty"`
		SiteAdmin         bool   `json:"site_admin,omitempty"`
	} `json:"author,omitempty"`
	NodeID          string        `json:"node_id,omitempty"`
	TagName         string        `json:"tag_name,omitempty"`
	TargetCommitish string        `json:"target_commitish,omitempty"`
	Name            string        `json:"name,omitempty"`
	Draft           bool          `json:"draft,omitempty"`
	Prerelease      bool          `json:"prerelease,omitempty"`
	CreatedAt       time.Time     `json:"created_at,omitempty"`
	PublishedAt     time.Time     `json:"published_at,omitempty"`
	Assets          []interface{} `json:"assets,omitempty"`
	TarballURL      string        `json:"tarball_url,omitempty"`
	ZipballURL      string        `json:"zipball_url,omitempty"`
	Body            string        `json:"body,omitempty"`
}

func OperatorLatestSupportedVersion() string {
	fetchOperatorVersionOnce.Do(func() {
		if operatorLatestVersion != "" {
			return
		}
		releases := fetchVersionsOrPanic(operatorReleasesUrl)
		operatorLatestVersion = selectLatestVersionOrPanic(releases)
	})

	return operatorLatestVersion
}

func SetOperatorLatestSupportedVersion(version string) {
	operatorLatestVersion = version
}

func ClusterOperatorLastVersion() string {
	fetchClusterOperatorVersionOnce.Do(func() {
		releases := fetchVersionsOrPanic(clusterOperatorReleasesUrl)
		clusterOperatorLastVersion = selectLatestVersionOrPanic(releases)
	})

	return clusterOperatorLastVersion
}

func fetchVersionsOrPanic(url string) []GithubRelease {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		panic(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	rawVersions, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	releases := []GithubRelease{}
	err = json.Unmarshal(rawVersions, &releases)
	if err != nil {
		panic(err)
	}

	return releases
}

func selectLatestVersionOrPanic(releases []GithubRelease) string {
	versions := []GithubRelease{}

	for _, release := range releases {
		if release.Draft {
			continue
		}
		if !enableUnofficialRelease && release.Prerelease {
			continue
		}

		if cleanupVersion(release.TagName) != "" {
			versions = append(versions, release)
		}
	}

	if len(versions) == 0 {
		panic("release not found")
	}

	sort.SliceStable(versions, func(i, j int) bool {
		versionI := cleanupVersion(versions[i].TagName)
		versionJ := cleanupVersion(versions[j].TagName)

		if versionI == versionJ {
			return versions[i].CreatedAt.After(versions[j].CreatedAt)
		}

		less, err := VersionIsLessThan(versionI, versionJ)
		if err != nil {
			panic(err)
		}
		return !less
	})

	return versions[0].TagName
}
