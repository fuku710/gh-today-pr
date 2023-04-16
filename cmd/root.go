package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/briandowns/spinner"
	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/gookit/color"
	"github.com/spf13/cobra"
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
	Title     string
	HtmlUrl   string `json:"html_url"`
	CreatedAt string `json:"created_at"`
}

var noTitle bool

var rootCmd = &cobra.Command{
	Use:   "gh-today-pr",
	Short: "list your commited Pull Request of the day.",
	Run: func(cmd *cobra.Command, args []string) {
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = color.Gray.Sprintf(" Fetching data...")
		s.Start()

		now := time.Now()

		client, err := gh.RESTClient(nil)
		if err != nil {
			log.Fatal(err)
		}

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

		pulls = SortPullRequests(pulls)

		s.Stop()
		color.Cyan.Printf("[%d/%d/%d] \n", now.Year(), now.Month(), now.Day())

		if len(pulls) == 0 {
			color.Danger.Println("No Pull Requests")
			return
		}
		for _, p := range pulls {
			if noTitle {
				fmt.Printf("%s\n", p.HtmlUrl)
			} else {
				fmt.Printf("%s\n |- %s\n\n", p.Title, p.HtmlUrl)
			}
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVar(&noTitle, "no-title", false, "no display title")
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
	reRef := regexp.MustCompile(`refs/heads/(.*)`)
	reMergeMessage := regexp.MustCompile(`Merge pull request`)

	eventMap := map[string]Event{}
	for _, e := range events {
		switch e.Type {
		case "PushEvent":
			payload := PushEventPayload{}
			json.Unmarshal(e.Payload, &payload)

			// Ignore merge commit
			commit := struct {
				Commit struct {
					Message string
				}
			}{}
			err := client.Get(fmt.Sprintf("repos/%s/commits/%s", e.Repo.Name, payload.Head), &commit)
			if err != nil {
				return nil, err
			}
			if reMergeMessage.MatchString(commit.Commit.Message) {
				continue
			}

			// Get default branch from repository of event
			repo := struct {
				DefeaultBranch string `json:"default_branch"`
			}{}
			err = client.Get(fmt.Sprintf("repos/%s", e.Repo.Name), &repo)
			if err != nil {
				return nil, err
			}

			branch := reRef.FindStringSubmatch(payload.Ref)[1]
			if branch != repo.DefeaultBranch {
				eventMap[branch] = e
			}
		case "CreateEvent":
			payload := CreateEventPayload{}
			json.Unmarshal(e.Payload, &payload)

			branch := payload.Ref
			if branch != payload.MasterBranch {
				eventMap[branch] = e
			}
		}
	}
	return eventMap, nil
}

func getPullRequests(client api.RESTClient, events map[string]Event) ([]PullRequest, error) {
	reRef := regexp.MustCompile(`refs/heads/(.*)`)
	reRepoName := regexp.MustCompile(`(.*)/(.*)`)

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
	today := time.Date(now.Year(), now.Month(), now.Day()-5, 0, 0, 0, 0, time.Local)
	return !target.Before(today)
}

func SortPullRequests(pulls []PullRequest) []PullRequest {
	sort.Slice(pulls, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339, pulls[i].CreatedAt)
		b, _ := time.Parse(time.RFC3339, pulls[j].CreatedAt)
		return a.Before(b)
	})
	return pulls
}
