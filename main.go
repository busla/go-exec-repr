package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"time"

	"dagger.io/dagger"
)

func getCommandOutput(bin string, args ...string) string {
	cmd := exec.Command(bin, args...)

	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	fmt.Print("out: ", output)
	s := string([]byte{57, 102, 102, 48, 55, 53, 100, 98})
	fmt.Println("bytestring: ", s)
	return string(output)
}

func getEnv(name string, value ...string) string {
	e := os.Getenv(name)
	if e == "" {
		if len(value) > 0 {
			return value[0]
		}
		panic(fmt.Sprintf("%s env missing!", name))
	}
	return e
}

func getCommand(c ...string) (string, []string) {
	return c[0], c[1:]
}

func getSha(name string) string {
	e := os.Getenv(name)
	bin, args := getCommand("git", "rev-parse", "--short=8", "HEAD")
	fmt.Println("args: ", args)
	if e == "" {
		currentUser, err := user.Current()
		if err != nil {
			panic(err)
		}
		currentTime := time.Now().Format("20060102150405")
		sha := getCommandOutput(bin, args...)

		out := fmt.Sprintf("%s-%s-%s", currentUser.Username, sha, currentTime)
		fmt.Println("out: ", out)
		return out

	}
	return e
}

func main() {
	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		panic(err)
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

	sha := getSha("COMMIT_SHA")

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
