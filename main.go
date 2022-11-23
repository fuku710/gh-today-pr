package main

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
)

type Event struct {
	Type string
	Repo struct {
		Name string
	}
	CreatedAt time.Time `json:"created_at"`
	Payload   json.RawMessage
}

type PushEventPayload struct {
	Ref     string
	Head    string
	Before  string
	Commits []struct {
		Url string
	}
}
type CreateEventPayload struct {
	Ref          string
	MasterBranch string `json:"master_branch"`
}

type PullRequest struct {
	Title   string
	HtmlUrl string `json:"html_url"`
}

func main() {
	client, err := gh.RESTClient(nil)
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()

	events, err := getEvents(client, now)
	if err != nil {
		log.Fatal(err)
	}

	eventMap, err := mapEvents(client, events)
	if err != nil {
		log.Fatal(err)
	}

	pulls, err := getPullRequests(client, eventMap)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("🌞 %d/%d/%d 🌝\n", now.Year(), now.Month(), now.Day())
	if len(pulls) == 0 {
		fmt.Println("No Pull Requests")
		return
	}
	for _, p := range pulls {
		fmt.Printf("%s(%s)\n", p.Title, p.HtmlUrl)
	}
}

func getEvents(client api.RESTClient, now time.Time) ([]Event, error) {
	user := struct{ Login string }{}
	err := client.Get("user", &user)
	if err != nil {
		return []Event{}, err
	}

	events := []Event{}

	for page := 1; ; page++ {
		res := []Event{}
		err = client.Get(fmt.Sprintf("users/%s/events?per_page=100&page=%d", user.Login, page), &res)
		if err != nil {
			return nil, err
		}

		for _, e := range res {
			// log.Printf("RepoName: %s,EventType: %s,CreatedAt: %s\n", e.Repo.Name, e.Type, e.CreatedAt.In(time.Local))
			if !IsToday(now, e.CreatedAt) {
				return events, nil
			}
			if e.Type == "PushEvent" || e.Type == "CreateEvent" {
				events = append(events, e)
			}
		}

		if len(events) < 100 {
			break
		}
	}

	return events, nil
}

func mapEvents(client api.RESTClient, events []Event) (map[string]Event, error) {
	eventMap := map[string]Event{}
	for _, e := range events {
		if e.Type == "PushEvent" {
			payload := PushEventPayload{}
			json.Unmarshal(e.Payload, &payload)

			// Get default branch from repository of event
			repo := struct {
				DefeaultBranch string `json:"default_branch"`
			}{}
			err := client.Get(fmt.Sprintf("repos/%s", e.Repo.Name), &repo)
			if err != nil {
				return nil, err
			}

			ref := fmt.Sprintf("refs/heads/%s", repo.DefeaultBranch)
			if payload.Ref != ref {
				eventMap[payload.Ref] = e
			}
		}
		if e.Type == "CreateEvent" {
			payload := CreateEventPayload{}
			json.Unmarshal(e.Payload, &payload)

			ref := fmt.Sprintf("refs/heads/%s", payload.MasterBranch)
			if payload.Ref != ref {
				eventMap[payload.Ref] = e
			}
		}
	}
	return eventMap, nil
}

func getPullRequests(client api.RESTClient, events map[string]Event) ([]PullRequest, error) {
	reRef := regexp.MustCompile("refs/heads/(.*)")
	reRepoName := regexp.MustCompile("(.*)/(.*)")

	pulls := []PullRequest{}
	for _, e := range events {
		payload := struct {
			Ref string
		}{}
		json.Unmarshal(e.Payload, &payload)
		var branch string
		if e.Type == "PushEvent" {
			branch = reRef.FindStringSubmatch(payload.Ref)[1]
		} else {
			branch = payload.Ref
		}
		matches := reRepoName.FindStringSubmatch(e.Repo.Name)[1:3]
		org := matches[0]
		repo := matches[1]
		head := org + ":" + branch

		ps := []PullRequest{}
		err := client.Get(fmt.Sprintf("repos/%s/%s/pulls?state=all&head=%s", org, repo, head), &ps)
		if err != nil {
			return []PullRequest{}, err
		}
		pulls = append(pulls, ps...)
	}

	return pulls, nil
}

func IsToday(now time.Time, target time.Time) bool {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	return !target.Before(today)
}
