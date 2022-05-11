package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/toast.v1"
)

type Configuration struct {
	ProjectId string
	TeamId    string
	SessionId string
}

type OpenSlot struct {
	End   string
	Start string
	Title string
	Id    int
	Ids   string
}

func main() {
	var refreshRate int
	var daysOffset int
	var newConfig string

	flag.IntVar(&refreshRate, "refresh", 120, "Time between refresh in seconds.")
	flag.IntVar(&daysOffset, "days", 2, "Range of days to look for available slots.")
	flag.Parse()

	var configuration Configuration

	fmt.Printf("-------------------------\n")
	fmt.Printf("-    42 Slot Watcher    -\n")
	fmt.Printf("-------------------------\n")

	_, err := os.Stat("config.json")
	if err != nil {
		getInfo(&configuration)
	} else {
		for {
			fmt.Print("Found previous config file. Create new config file? [Y/n] ")
			fmt.Scanln(&newConfig)
			if len(newConfig) > 0 && newConfig[0] == 'Y' {
				getInfo(&configuration)
				break
			} else if len(newConfig) > 0 && newConfig[0] == 'n' {
				break
			}
		}
	}
	content, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatalln(err)
	}
	err = json.Unmarshal(content, &configuration)
	if err != nil {
		log.Fatalln(err)
	}

	notice := toast.Notification{
		AppID:   "42 Slot Watcher",
		Title:   "Slot(s) Available",
		Message: fmt.Sprintf("Found available slot(s) for project %s", configuration.ProjectId),
		Actions: []toast.Action{
			{"protocol", "Open Page", fmt.Sprintf("https://projects.intra.42.fr/projects/%s/slots?team_id=%s", configuration.ProjectId, configuration.TeamId)},
		},
	}

	var openSlot []OpenSlot
	var slots []byte

	fmt.Printf("Project ID: %s Team ID: %v\n", configuration.ProjectId, configuration.TeamId)

	for {
		slots = getSlots(daysOffset, configuration.ProjectId, configuration.TeamId, configuration.SessionId)
		err := json.Unmarshal(slots, &openSlot)
		if err != nil {
			log.Fatalln(err)
		}
		if len(openSlot) > 0 {
			log.Println("Found available slots.")
			err := notice.Push()
			if err != nil {
				log.Fatalln(err)
			}
		} else {
			log.Println("No slot available.")
		}
		time.Sleep(time.Duration(refreshRate) * time.Second)
	}
}

func getInfo(configuration *Configuration) {
	fmt.Println("Please enter information")
	fmt.Print("Project ID: ")
	fmt.Scanln(&configuration.ProjectId)
	fmt.Print("Team ID: ")
	fmt.Scanln(&configuration.TeamId)
	fmt.Print("Session ID: ")
	fmt.Scanln(&configuration.SessionId)
	config, err := os.OpenFile("config.json", os.O_CREATE, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	config.Close()
	content, err := json.Marshal(configuration)
	if err != nil {
		log.Fatalln(err)
	}
	err = ioutil.WriteFile("config.json", content, 0644)
}

func getSlots(daysOffset int, projectId, teamId, sessionId string) []byte {
	currentTime := time.Now()
	startDate := currentTime.Format("2006-02-05")
	currentTime = currentTime.AddDate(0, 0, daysOffset)
	endDate := currentTime.Format("2006-02-05")

	client := &http.Client{}
	url := fmt.Sprintf("https://projects.intra.42.fr/projects/%s/slots.json?team_id=%s&start=%s&end=%s", projectId, teamId, startDate, endDate)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("cookie", fmt.Sprintf("_intra_42_session_production=%s", sessionId))
	req.Host = "projects.intra.42.fr"
	res, err := client.Do(req)
	if err != nil {
		log.Panic(err)
	}
	defer res.Body.Close()

	var resBody []byte
	if res.StatusCode == http.StatusOK {
		resBody, err = io.ReadAll(res.Body)
		if err != nil {
			log.Println(err)
		}
	} else {
		log.Fatalf("%v Response from server\n", res.StatusCode)
	}
	return resBody
}
