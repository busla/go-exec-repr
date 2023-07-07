package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"

	"dagger.io/dagger"
	"github.com/go-cmd/cmd"
)

func getCommandOutputWithGoCmd(bin string, args ...string) string {
	c := cmd.NewCmd(bin, args...)
	status := <-c.Start()
	out := strings.Join(status.Stdout, " ")
	return out
}

func getCommandOutput(bin string, args ...string) string {
	c := exec.Command(bin, args...)
	stdout, err := c.Output()

	if err != nil {
		log.Fatalf(err.Error())
	}
	return string(stdout)

}

func getEnv(name string, value ...string) string {
	e := os.Getenv(name)
	if e == "" {
		if len(value) > 0 {
			return value[0]
		}
		log.Fatalf(fmt.Sprintf("%s env missing!", name))
	}
	return e
}

func getSha(name string) string {
	e := os.Getenv(name)
	useGoCmd := os.Getenv("USE_GO_CMD")
	var sha string
	if e == "" {
		currentUser, err := user.Current()
		currentTime := time.Now().Format("20060102150405")

		if err != nil {
			log.Fatalf(err.Error())
		}
		if useGoCmd == "1" {
			sha = getCommandOutputWithGoCmd("git", "rev-parse", "--short=8", "HEAD")
			return fmt.Sprintf("%s-%s-%s", currentUser.Username, sha, currentTime)

		} else {
			sha = getCommandOutput("git", "rev-parse", "--short=8", "HEAD")
			return fmt.Sprintf("%s-%s-%s", currentUser.Username, sha, currentTime)
		}
	}
	return e
}

func main() {
	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	baseImage := getEnv("BASE_IMAGE", "node:lts-alpine3.18")
	node := client.Container().
		From(baseImage).
		WithExec([]string{"node", "--version"})

	base := node.Pipeline("base").
		WithEnvVariable("CI", "true").
		WithEntrypoint(nil).
		WithWorkdir("/app")

	buildDir := base.Pipeline("install").
		WithEntrypoint(nil).WithExec([]string{"/bin/sh", "-c", `
		echo $(node --version) > node-version.txt
	`}).Directory("/app")

	sha := getSha("GIT_SHA")
	fmt.Println("sha is: ", sha)

	ref, err := client.Container().
		From(baseImage).
		WithWorkdir("/app").
		WithDirectory("/app", buildDir).
		Publish(ctx, fmt.Sprintf("ttl.sh/dagger-exec-repr:%s", sha))
	if err != nil {
		panic(err)
	}
	fmt.Println("Published at:", ref)
}
