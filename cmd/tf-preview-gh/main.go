package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/ffddorf/tf-preview-github/pkg/quicpunch"
	"github.com/google/go-github/v57/github"
)

type countingReader struct {
	io.Reader
	readBytes int
}

func (c *countingReader) Read(dst []byte) (int, error) {
	n, err := c.Reader.Read(dst)
	c.readBytes += n
	return n, err
}

var ignoredGroupNames = []string{
	"Operating System",
	"Runner Image",
	"Runner Image Provisioner",
	"GITHUB_TOKEN Permissions",
}

func streamLogs(logsURL *url.URL, skip int64) (int64, error) {
	logs, err := http.Get(logsURL.String())
	if err != nil {
		return 0, err
	}
	if logs.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("invalid status for logs: %d", logs.StatusCode)
	}
	defer logs.Body.Close()

	if _, err := io.Copy(io.Discard, io.LimitReader(logs.Body, skip)); err != nil {
		return 0, err
	}

	r := &countingReader{Reader: logs.Body}
	scanner := bufio.NewScanner(r)
	groupDepth := 0
	for scanner.Scan() {
		line := scanner.Text()
		ts, rest, ok := strings.Cut(line, " ")
		if !ok {
			rest = ts
		}
		if groupName, ok := strings.CutPrefix(rest, "##[group]"); ok {
			groupDepth++
			if !slices.Contains(ignoredGroupNames, groupName) {
				fmt.Printf("\n# %s\n", groupName)
			}
		}
		if groupDepth == 0 {
			fmt.Println(rest)
		}
		if strings.HasPrefix(rest, "##[endgroup]") {
			groupDepth--
		}
	}
	if err := scanner.Err(); err != nil {
		return int64(r.readBytes), err
	}

	return int64(r.readBytes), err
}

var (
	owner            string
	repo             string
	workflowRef      string
	workflowFilename string
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	flag.StringVar(&owner, "github-owner", "ffddorf", "Repository owner")
	flag.StringVar(&repo, "github-repo", "", "Repository name")
	flag.StringVar(&workflowFilename, "workflow-file", "preview.yaml", "Name of the workflow file to run for previews")
	flag.StringVar(&workflowRef, "workflow-ref", "main", "Ref to run the workflow from")
	flag.Parse()

	if repo == "" {
		panic("Missing flag: -github-repo")
	}

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	sess, serve, err := quicpunch.ServeWorkspace(ctx, cwd)
	if err != nil {
		panic(err)
	}
	go func() {
		err := serve(ctx)
		if err != nil {
			fmt.Printf("workspace server failed: %v", err)
		}
	}()

	// steal token from GH CLI
	cmd := exec.CommandContext(ctx, "gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	token := strings.TrimSpace(string(out))
	gh := github.NewClient(nil).WithAuthToken(token)

	startedAt := time.Now().UTC()

	// start workflow
	_, err = gh.Actions.CreateWorkflowDispatchEventByFileName(ctx,
		owner, repo, workflowFilename,
		github.CreateWorkflowDispatchEventRequest{
			Ref: workflowRef,
			Inputs: map[string]interface{}{
				"webrtc-session": sess,
			},
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Println("Waiting for run to start...")

	// find workflow run
	var run *github.WorkflowRun
	err = backoff.Retry(func() error {
		workflows, _, err := gh.Actions.ListWorkflowRunsByFileName(
			ctx, owner, repo, workflowFilename,
			&github.ListWorkflowRunsOptions{
				Created: fmt.Sprintf(">=%s", startedAt.Format("2006-01-02T15:04")),
				Branch:  workflowRef,
			},
		)
		if err != nil {
			return backoff.Permanent(err)
		}
		if len(workflows.WorkflowRuns) == 0 {
			return fmt.Errorf("no workflow runs found")
		}

		run = workflows.WorkflowRuns[0]
		return nil
	}, backoff.NewExponentialBackOff())
	if err != nil {
		panic(err)
	}

	var jobID int64
	err = backoff.Retry(func() error {
		jobs, _, err := gh.Actions.ListWorkflowJobs(ctx,
			owner, repo, *run.ID,
			&github.ListWorkflowJobsOptions{},
		)
		if err != nil {
			return backoff.Permanent(err)
		}
		if len(jobs.Jobs) == 0 {
			return fmt.Errorf("no jobs found")
		}

		jobID = *jobs.Jobs[0].ID
		return nil
	}, backoff.NewExponentialBackOff())
	if err != nil {
		panic(err)
	}

	logsURL, _, err := gh.Actions.GetWorkflowJobLogs(ctx, owner, repo, jobID, 2)
	if err != nil {
		panic(err)
	}

	var readBytes int64
	for {
		n, err := streamLogs(logsURL, readBytes)
		if err != nil {
			panic(err)
		}
		readBytes += n

		// check if job is done
		job, _, err := gh.Actions.GetWorkflowJobByID(ctx, owner, repo, jobID)
		if err != nil {
			panic(err)
		}
		if job.CompletedAt != nil {
			fmt.Println("Job complete.")
			break
		}
	}
}
