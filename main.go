package main

import (
	"encoding/json"
	"fmt"
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

type PushEvent struct {
	Type string
	Repo struct {
		Name string
	}
	CreatedAt time.Time `json:"created_at"`
	Payload   struct {
		Ref     string
		Head    string
		Before  string
		Commits []struct {
			Url string
		}
	}
}

type PullRequest struct {
	Title   string
	HtmlUrl string `json:"html_url"`
}

func main() {
	fmt.Printf("Today's PullRequests!\n\n")

	client, err := gh.RESTClient(nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	now := time.Now()

	events, err := getPushEventsIn24hors(client, now)
	if err != nil {
		fmt.Println(err)
		return
	}

	pulls, err := getPushedPullRequests(client, events)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, p := range pulls {
		fmt.Printf("%s(%s)\n", p.Title, p.HtmlUrl)
	}
}

func getPushEventsIn24hors(client api.RESTClient, now time.Time) ([]PushEvent, error) {
	user := struct{ Login string }{}
	err := client.Get("user", &user)
	if err != nil {
		return []PushEvent{}, err
	}

	events := []Event{}
	err = client.Get(fmt.Sprintf("users/%s/events", user.Login), &events)
	if err != nil {
		return []PushEvent{}, err
	}

	pushEvents := []PushEvent{}
	for _, e := range events {
		if !IsToday(now, e.CreatedAt) {
			break
		}
		if e.Type == "PushEvent" {
			pushEvent := PushEvent{
				Type: e.Type,
				Repo: e.Repo,
			}
			json.Unmarshal(e.Payload, &pushEvent.Payload)
			pushEvents = append(pushEvents, pushEvent)
		}
	}

	return pushEvents, nil
}

func getPushedPullRequests(client api.RESTClient, events []PushEvent) ([]PullRequest, error) {
	reRef := regexp.MustCompile("refs/heads/(.*)")
	reRepoName := regexp.MustCompile("(.*)/(.*)")

	pulls := []PullRequest{}
	for _, e := range events {
		branch := reRef.FindStringSubmatch(e.Payload.Ref)[1]
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
