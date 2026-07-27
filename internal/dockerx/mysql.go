package dockerx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

const mysqlContainerName = "mysql"

var databaseNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)

// DropSiteDatabase mirrors kejilion.sh web_del for a validated domain while
// keeping the root password out of command arguments, logs, and API responses.
func (c *Client) DropSiteDatabase(ctx context.Context, domain string) (bool, error) {
	database := siteDatabaseName(domain)
	if !databaseNamePattern.MatchString(database) {
		return false, errors.New("site database name is invalid or too long")
	}
	raw, err := c.inspect(ctx, mysqlContainerName)
	if err != nil {
		if isDockerStatus(err, http.StatusNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("inspect fixed MySQL container: %w", err)
	}
	if strings.TrimPrefix(raw.Name, "/") != mysqlContainerName ||
		!dockerExecIDPattern.MatchString(raw.ID) ||
		!raw.State.Running || raw.State.Status != "running" ||
		raw.State.Paused || raw.State.Restarting {
		return false, errors.New("fixed MySQL container is not running normally")
	}
	password := containerEnvironmentValue(raw.Config.Env, "MYSQL_ROOT_PASSWORD")
	if password == "" {
		password = containerEnvironmentValue(raw.Config.Env, "MARIADB_ROOT_PASSWORD")
	}
	if password == "" || len(password) > 4096 || strings.ContainsAny(password, "\r\n\x00") {
		return false, errors.New("MySQL root credential is unavailable or unsafe")
	}
	defer clearString(&password)

	command := []string{
		"mysql",
		"-u",
		"root",
		"-N",
		"-B",
		"-e",
		"SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME='" +
			database +
			"'; DROP DATABASE IF EXISTS `" +
			database +
			"`;",
	}
	execID, err := c.createFixedExec(
		ctx,
		raw.ID,
		command,
		[]string{"MYSQL_PWD=" + password},
	)
	if err != nil {
		return false, err
	}
	output, truncated, err := c.startNginxExec(ctx, execID)
	if err != nil {
		return false, err
	}
	state, err := c.inspectNginxExec(ctx, execID)
	if err != nil {
		return false, err
	}
	if state.Running {
		return false, ErrNginxExecRunning
	}
	if state.ExitCode != 0 {
		safeOutput, _ := boundedRedactedNginxOutput(output, truncated)
		return false, fmt.Errorf("drop site database failed: %s", safeOutput)
	}
	plain := strings.TrimSpace(string(demuxNginxExecStream(output)))
	return plain == database, nil
}

func (c *Client) createFixedExec(
	ctx context.Context,
	containerID string,
	command []string,
	environment []string,
) (string, error) {
	if !dockerExecIDPattern.MatchString(containerID) || len(command) == 0 {
		return "", ErrInvalidDockerExec
	}
	payload, err := json.Marshal(struct {
		AttachStdout bool     `json:"AttachStdout"`
		AttachStderr bool     `json:"AttachStderr"`
		Tty          bool     `json:"Tty"`
		Cmd          []string `json:"Cmd"`
		Env          []string `json:"Env,omitempty"`
	}{
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
		Cmd:          command,
		Env:          environment,
	})
	if err != nil {
		return "", err
	}
	data, _, err := c.nginxDockerRequest(
		ctx,
		http.MethodPost,
		"/containers/"+containerID+"/exec",
		payload,
		4<<10,
	)
	if err != nil {
		return "", err
	}
	var response struct {
		ID string `json:"Id"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return "", err
	}
	if !dockerExecIDPattern.MatchString(response.ID) {
		return "", ErrInvalidDockerExec
	}
	return response.ID, nil
}

func siteDatabaseName(domain string) string {
	return strings.Map(func(value rune) rune {
		if value >= 'A' && value <= 'Z' ||
			value >= 'a' && value <= 'z' ||
			value >= '0' && value <= '9' {
			return value
		}
		return '_'
	}, domain)
}

func containerEnvironmentValue(values []string, key string) string {
	prefix := key + "="
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return strings.TrimPrefix(value, prefix)
		}
	}
	return ""
}

func clearString(value *string) {
	if value == nil {
		return
	}
	*value = ""
}
