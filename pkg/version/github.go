package version

import (
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/storageos/kubectl-storageos/pkg/consts"
	pluginutils "github.com/storageos/kubectl-storageos/pkg/utils"
)

const (
	operatorReleasesUrl     = "https://api.github.com/repos/storageos/operator/releases"
	etcdOperatorReleasesUrl = "https://api.github.com/repos/storageos/etcd-cluster-operator/releases"
	// TODO: No release exists for portal-manager yet
	// portalManagerReleasesUrl   = "https://api.github.com/repos/storageos/portal-manager/releases"
)

var (
	operatorLatestVersion      string
	etcdOperatorLatestVersion  string
	portalManagerLatestVersion string

	fetchOperatorVersionOnce     = sync.Once{}
	fetchEtcdOperatorVersionOnce = sync.Once{}
	// TODO: No release exists for portal-manager yet
	// fetchPortalManagerVersionOnce   = sync.Once{}
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

func EtcdOperatorLatestSupportedVersion() string {
	fetchEtcdOperatorVersionOnce.Do(func() {
		if etcdOperatorLatestVersion != "" {
			return
		}
		releases := fetchVersionsOrPanic(etcdOperatorReleasesUrl)
		etcdOperatorLatestVersion = selectLatestVersionOrPanic(releases)
	})

	return etcdOperatorLatestVersion
}

func PortalManagerLatestSupportedVersion() string {
	// TODO: No release exists for portal-manager yet
	/*
		fetchPortalManagerVersionOnce.Do(func() {
			if portalManagerLatestVersion != "" {
				return
			}
			releases := fetchVersionsOrPanic(portalManagerReleasesUrl)
			portalManagerLatestVersion = selectLatestVersionOrPanic(releases)
		})

		return portalManagerLatestVersion
	*/
	return portalManagerLatestVersion
}

func LocalPathProvisionerLatestSupportVersion() string {
	// Pin this version and only update when we've checked the latest version works with storageos
	return "https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.22/deploy/local-path-storage.yaml"
}

func SetOperatorLatestSupportedVersion(version string) {
	operatorLatestVersion = version
}

func SetEtcdOperatorLatestSupportedVersion(version string) {
	etcdOperatorLatestVersion = version
}

func SetPortalManagerLatestSupportedVersion(version string) {
	portalManagerLatestVersion = version
}

func ClusterOperatorLastVersion() string {
	return consts.ClusterOperatorLastVersion
}

func fetchVersionsOrPanic(url string) []GithubRelease {
	rawVersions, err := pluginutils.FetchHttpContent(url, nil)
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
