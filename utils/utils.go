package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudfoundry/cli/plugin"
)

type Utils interface {
	GetAppGuid(string) (string, error, []string)
	GetTargetSpace() (string, error, []string)
	GetSpaceGuid(string) (string, error, []string)
	FindDomain() (string, error, []string)
	StartApp(string) ([]string, error)
	CreateRoute(string, string, string) ([]string, error)
	MapRoute(string, string, string) ([]string, error)
	GetHealthCheck(string) (string, []string, error)
	DetachAppRoutes(string) ([]string, error)
	UpdateApp(string, string, string) ([]string, error)
}

type utils struct {
	cli plugin.CliConnection
}

type AppSummary struct {
	HealthCheckType string  `json:"health_check_type"`
	Routes          []Route `json:"routes"`
}
type Route struct {
	Guid string `json:"guid,omitempty"`
}

func NewUtils(cli plugin.CliConnection) Utils {
	return &utils{
		cli: cli,
	}
}

func (u *utils) GetAppGuid(appName string) (string, error, []string) {
	result, err := u.cli.CliCommandWithoutTerminalOutput("app", appName, "--guid")
	if err != nil {
		if strings.Contains(result[0], "FAILED") {
			return "", errors.New("App " + appName + " not found."), result
		}

		return "", err, result
	}

	return strings.TrimSpace(result[0]), nil, []string{}
}

func (u *utils) GetTargetSpace() (string, error, []string) {
	output, err := u.cli.CliCommandWithoutTerminalOutput("target")
	if err != nil {
		return "", err, output
	}

	if len(output) < 5 {
		return "", errors.New("Currently not targeted"), output
	}

	if strings.HasPrefix(output[4], "Space:") && !strings.Contains(output[4], "No space targeted") {
		space := strings.TrimPrefix(strings.TrimSpace(output[4]), "Space:")
		return strings.TrimSpace(space), nil, []string{}
	} else {
		return "", errors.New("Currently not targeted"), output
	}
	return "", nil, output
}

func (u *utils) GetSpaceGuid(spaceName string) (string, error, []string) {
	output, err := u.cli.CliCommandWithoutTerminalOutput("space", spaceName, "--guid")
	if err != nil {
		if strings.Contains(output[0], "FAILED") {
			return "", errors.New("Getting space guid..."), output
		}

		return "", err, output
	}
	return strings.TrimSpace(output[0]), nil, []string{}
}

func (u *utils) CreateRoute(space, domain, host string) ([]string, error) {
	return u.cli.CliCommandWithoutTerminalOutput("create-route", space, domain, "-n", host)
}

func (u *utils) FindDomain() (string, error, []string) {
	output, err := u.cli.CliCommandWithoutTerminalOutput("domains")
	if err != nil {
		return "", err, output
	}

	if len(output) < 3 {
		return "", errors.New("No domain available"), output
	}

	domain := strings.TrimSuffix(strings.TrimSpace(output[2]), "shared")
	domain = strings.TrimSuffix(domain, "owned")

	return strings.TrimSpace(domain), nil, []string{}
}

func (u *utils) MapRoute(appName, domain, host string) ([]string, error) {
	return u.cli.CliCommandWithoutTerminalOutput("map-route", appName, domain, "-n", host)
}

func (u *utils) StartApp(appName string) ([]string, error) {
	return u.cli.CliCommand("start", appName)
}

func (u *utils) GetHealthCheck(appGuid string) (string, []string, error) {
	output, err := u.cli.CliCommandWithoutTerminalOutput("curl", "/v2/apps/"+appGuid+"/summary")
	if err != nil {
		return "", output, err
	}

	if !strings.Contains(output[0], `"health_check_type": `) {
		return "", output, errors.New(fmt.Sprintf("%s\nJSON:\n", "'health_check_type' flag is not found in json response"))
	}

	b := []byte(output[0])
	summary := AppSummary{}
	err = json.Unmarshal(b, &summary)
	if err != nil {
		return "", output, err
	}

	return summary.HealthCheckType, []string{}, nil
}

func (u *utils) DetachAppRoutes(appGuid string) ([]string, error) {
	output, err := u.cli.CliCommandWithoutTerminalOutput("curl", "/v2/apps/"+appGuid+"/summary")
	if err != nil {
		return output, err
	}

	b := []byte(output[0])
	summary := AppSummary{}
	err = json.Unmarshal(b, &summary)
	if err != nil {
		return output, err
	}

	for _, route := range summary.Routes {
		if output, err = u.cli.CliCommandWithoutTerminalOutput("curl", "/v2/routes/"+route.Guid+"/apps/"+appGuid, "-X", "DELETE"); err != nil {
			return output, err
		}
	}

	return nil, nil
}

func (u *utils) UpdateApp(appGuid, field, value string) ([]string, error) {
	return u.cli.CliCommandWithoutTerminalOutput("curl", "/v2/apps/"+appGuid, "-X", "PUT", "-d", `{"`+field+`":"`+value+`"}`)
}
