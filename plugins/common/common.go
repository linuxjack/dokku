package common

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	sh "github.com/codeskyblue/go-sh"
)

// ShellCmd represents a shell command to be run for dokku
type ShellCmd struct {
	Env           map[string]string
	Command       *exec.Cmd
	CommandString string
	Args          []string
	ShowOutput    bool
}

// NewShellCmd returns a new ShellCmd struct
func NewShellCmd(command string) *ShellCmd {
	items := strings.Split(command, " ")
	cmd := items[0]
	args := items[1:]
	return &ShellCmd{
		Command:       exec.Command(cmd, args...),
		CommandString: command,
		Args:          args,
		ShowOutput:    true,
	}
}

// Execute is a lightweight wrapper around exec.Command
func (sc *ShellCmd) Execute() bool {
	env := os.Environ()
	for k, v := range sc.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	sc.Command.Env = env
	if sc.ShowOutput {
		sc.Command.Stdout = os.Stdout
		sc.Command.Stderr = os.Stderr
	}
	err := sc.Command.Run()
	if err != nil {
		return false
	}
	return true
}

// VerifyAppName verifies app name format and app existence"
func VerifyAppName(appName string) (err error) {
	dokkuRoot := MustGetEnv("DOKKU_ROOT")
	appRoot := strings.Join([]string{dokkuRoot, appName}, "/")
	_, err = os.Stat(appRoot)
	if os.IsNotExist(err) {
		return fmt.Errorf("App %s does not exist: %v\n", appName, err)
	}
	r, _ := regexp.Compile("^[a-z].*")
	if !r.MatchString(appName) {
		return fmt.Errorf("App name (%s) must begin with lowercase alphanumeric character\n", appName)
	}
	return err
}

// MustGetEnv returns env variable or fails if it's not set
func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		LogFail(fmt.Sprintf("%s not set!", key))
	}
	return value
}

// LogFail is the failure log formatter
// prints text to stderr and exits with status 1
func LogFail(text string) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf("FAILED: %s", text))
	os.Exit(1)
}

// GetDeployingAppImageName returns deploying image identifier for a given app, tag tuple. validate if tag is presented
func GetDeployingAppImageName(appName, imageTag, imageRepo string) (imageName string) {
	if appName == "" {
		LogFail("(GetDeployingAppImageName) APP must not be empty")
	}

	b, err := sh.Command("plugn", "trigger", "deployed-app-repository", appName).Output()
	if err != nil {
		LogFail(err.Error())
	}
	imageRemoteRepository := string(b[:])

	b, err = sh.Command("plugn", "trigger", "deployed-app-image-tag", appName).Output()
	if err != nil {
		LogFail(err.Error())
	}
	newImageTag := string(b[:])

	b, err = sh.Command("plugn", "trigger", "deployed-app-image-repo", appName).Output()
	if err != nil {
		LogFail(err.Error())
	}
	newImageRepo := string(b[:])

	if newImageRepo != "" {
		imageRepo = newImageRepo
	}
	if newImageTag != "" {
		imageTag = newImageTag
	}
	if imageRepo == "" {
		imageRepo = GetAppImageRepo(appName)
	}
	if imageTag == "" {
		imageTag = "latest"
	}

	imageName = fmt.Sprintf("%s%s:%s", imageRemoteRepository, imageRepo, imageTag)
	if !VerifyImage(imageName) {
		LogFail(fmt.Sprintf("app image (%s) not found", imageName))
	}
	return
}

// GetAppImageRepo is the central definition of a dokku image repo pattern
func GetAppImageRepo(appName string) string {
	return strings.Join([]string{"dokku", appName}, "/")
}

// VerifyImage returns true if docker image exists in local repo
func VerifyImage(image string) bool {
	imageCmd := NewShellCmd(strings.Join([]string{"docker inspect", image}, " "))
	imageCmd.ShowOutput = false
	if imageCmd.Execute() {
		return true
	}
	return false
}
