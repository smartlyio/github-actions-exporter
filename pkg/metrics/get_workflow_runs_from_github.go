package metrics

import (
	"context"
	"github-actions-exporter/pkg/config"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v38/github"
)

// getFieldValue return value from run element which corresponds to field
func getFieldValue(repo string, run github.WorkflowRun, field string) string {
	switch field {
	case "repo":
		return repo
	case "id":
		return strconv.FormatInt(*run.ID, 10)
	case "node_id":
		return *run.NodeID
	case "head_branch":
		return *run.HeadBranch
	case "head_sha":
		return *run.HeadSHA
	case "run_number":
		return strconv.Itoa(*run.RunNumber)
	case "workflow_id":
		return strconv.FormatInt(*run.WorkflowID, 10)
	case "workflow":
		return *run.Name
	case "event":
		return *run.Event
	case "status":
		return *run.Status
	}
	return ""
}

//
func getRelevantFields(repo string, run *github.WorkflowRun) []string {
	relevantFields := strings.Split(config.WorkflowFields, ",")
	result := make([]string, len(relevantFields))
	for i, field := range relevantFields {
		result[i] = getFieldValue(repo, *run, field)
	}
	return result
}


func getRelevantJobFields(repo string, run *github.WorkflowRun, job *github.WorkflowJob) []string {
	relevantFields := strings.Split(config.JobFields, ",")
	result := make([]string, len(relevantFields))
	for i, field := range relevantFields {
		switch field {
			case "id":
			result[i] = strconv.FormatInt(*job.ID, 10)
			case "name":
			result[i] = *job.Name
			case "node_id":
			result[i] = *job.NodeID
			case "head_sha":
			result[i] = *job.HeadSHA
			case "status":
			result[i] = *job.Status
			case "conclusion":
			result[i] = *job.Conclusion
			// case "runner_id":
			// result[i] = strconv.FormatInt(*job.RunnerID, 10)
			// case "runner_name":
			// result[i] = *job.RunnerName
			case "run_id":
			result[i] = strconv.FormatInt(*run.ID, 10)
			default:
			result[i] = getFieldValue(repo, *run, field)
		}
	}
	return result
}


func getWorkflowRunJobsFromGithub(repo stirng, run *github.WorkflowRun) {
	r := strings.Split(repo, "/")
	runId := getFieldValue(repo, *run, "id")
	resp, _, err := client.Actions.ListWorkflowJobs(context.Background(), r[0], r[1], runId, nil)
	if err != nil {
		log.Printf("ListWorkflowJobs error for %s: %s", repo, err.Error())
	} else {
		for _, job := range resp.Jobs {
			fields := getRelevantJobFields(repo, run, *job)

			created := job.CreatedAt.Time.Unix()
			updated := job.UpdatedAt.Time.Unix()
			elapsed := updated - created
			workflowRunJobDurationGauge.WithLabelValues(fields...).Set(float64(elapsed * 1000))
		}
	}
}

// getWorkflowRunsFromGithub - return informations and status about a workflow
func getWorkflowRunsFromGithub() {
	for {
		for _, repo := range config.Github.Repositories.Value() {
			r := strings.Split(repo, "/")
			resp, _, err := client.Actions.ListRepositoryWorkflowRuns(context.Background(), r[0], r[1], nil)
			if err != nil {
				log.Printf("ListRepositoryWorkflowRuns error for %s: %s", repo, err.Error())
			} else {
				for _, run := range resp.WorkflowRuns {
					var s float64 = 0
					if run.GetConclusion() == "success" {
						s = 1
					} else if run.GetConclusion() == "skipped" {
						s = 2
					} else if run.GetConclusion() == "in_progress" {
						s = 3
					} else if run.GetConclusion() == "queued" {
						s = 4
					}

					fields := getRelevantFields(repo, run)

					workflowRunStatusGauge.WithLabelValues(fields...).Set(s)

					resp, _, err := client.Actions.GetWorkflowRunUsageByID(context.Background(), r[0], r[1], *run.ID)
					if err != nil { // Fallback for Github Enterprise
						created := run.CreatedAt.Time.Unix()
						updated := run.UpdatedAt.Time.Unix()
						elapsed := updated - created
						workflowRunDurationGauge.WithLabelValues(fields...).Set(float64(elapsed * 1000))
					} else {
						workflowRunDurationGauge.WithLabelValues(fields...).Set(float64(resp.GetRunDurationMS()))
					}

					getWorkflowRunJobsFromGithub(repo, *run)
				}
			}
		}

		time.Sleep(time.Duration(config.Github.Refresh) * time.Second)
	}
}
