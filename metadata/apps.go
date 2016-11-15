package metadata

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry/cli/plugin"
	"github.com/kkellner/cloudfoundry-top-plugin/debug"
)

const MEGABYTE = (1024 * 1024)

type AppResponse struct {
	Count     int           `json:"total_results"`
	Pages     int           `json:"total_pages"`
	NextUrl   string        `json:"next_url"`
	Resources []AppResource `json:"resources"`
}

type AppResource struct {
	Meta   Meta `json:"metadata"`
	Entity App  `json:"entity"`
}

type App struct {
	Guid      string `json:"guid"`
	Name      string `json:"name,omitempty"`
	SpaceGuid string `json:"space_guid,omitempty"`
	SpaceName string
	OrgGuid   string
	OrgName   string

	StackGuid   string  `json:"stack_guid,omitempty"`
	MemoryMB    float64 `json:"memory,omitempty"`
	DiskQuotaMB float64 `json:"disk_quota,omitempty"`

	Environment map[string]interface{} `json:"environment_json,omitempty"`
	Instances   float64                `json:"instances,omitempty"`
	State       string                 `json:"state,omitempty"`
	EnableSsh   bool                   `json:"enable_ssh,omitempty"`

	PackageState        string `json:"package_state,omitempty"`
	StagingFailedReason string `json:"staging_failed_reason,omitempty"`
	StagingFailedDesc   string `json:"staging_failed_description,omitempty"`
	DetectedStartCmd    string `json:"detected_start_command,omitempty"`
	//DockerCredentials string  `json:"docker_credentials_json,omitempty"`
	//audit.app.create event fields
	Console           bool   `json:"console,omitempty"`
	Buildpack         string `json:"buildpack,omitempty"`
	DetectedBuildpack string `json:"detected_buildpack,omitempty"`

	HealthcheckType    string  `json:"health_check_type,omitempty"`
	HealthcheckTimeout float64 `json:"health_check_timeout,omitempty"`
	Production         bool    `json:"production,omitempty"`
	//app.crash event fields
	//Index           float64 `json:"index,omitempty"`
	//ExitStatus      string  `json:"exit_status,omitempty"`
	//ExitDescription string  `json:"exit_description,omitempty"`
	//ExitReason      string  `json:"reason,omitempty"`

}

var (
	appsMetadataCache         []App
	totalMemoryAllStartedApps float64
	totalDiskAllStartedApps   float64
)

func AppMetadataSize() int {
	return len(appsMetadataCache)
}

func AllApps() []App {
	return appsMetadataCache
}

func FindAppMetadata(appId string) App {
	// TODO: put this into a map for efficiency
	for _, app := range appsMetadataCache {
		if app.Guid == appId {
			return app
		}
	}
	return App{}
}

func GetTotalMemoryAllStartedApps() float64 {
	if totalMemoryAllStartedApps == 0 {
		for _, app := range appsMetadataCache {
			if app.State == "STARTED" {
				totalMemoryAllStartedApps = totalMemoryAllStartedApps + ((app.MemoryMB * MEGABYTE) * app.Instances)
			}
		}
	}
	return totalMemoryAllStartedApps
}

func GetTotalDiskAllStartedApps() float64 {
	if totalDiskAllStartedApps == 0 {
		for _, app := range appsMetadataCache {
			if app.State == "STARTED" {
				totalDiskAllStartedApps = totalDiskAllStartedApps + ((app.DiskQuotaMB * MEGABYTE) * app.Instances)
			}
		}
	}
	return totalDiskAllStartedApps
}

func LoadAppCache(cliConnection plugin.CliConnection) {
	data, err := getAppMetadata(cliConnection)
	if err != nil {
		debug.Warn(fmt.Sprintf("*** app metadata error: %v\n", err.Error()))
		return
	}
	appsMetadataCache = data
}

func getAppMetadata(cliConnection plugin.CliConnection) ([]App, error) {

	url := "/v2/apps"
	appsMetadata := []App{}

	handleRequest := func(outputBytes []byte) (interface{}, error) {
		var appResp AppResponse
		err := json.Unmarshal(outputBytes, &appResp)
		if err != nil {
			debug.Warn(fmt.Sprintf("*** %v unmarshal parsing output: %v\n", url, string(outputBytes[:])))
			return appsMetadata, err
		}
		for _, app := range appResp.Resources {
			app.Entity.Guid = app.Meta.Guid
			appsMetadata = append(appsMetadata, app.Entity)
		}
		return appResp, nil
	}

	callRetriableAPI(cliConnection, url, handleRequest)
	// Flush the total counters
	totalMemoryAllStartedApps = 0
	totalDiskAllStartedApps = 0
	return appsMetadata, nil

}
