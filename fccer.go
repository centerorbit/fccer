package main

import (
	"fmt"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

type entry struct {
	Number    string            `json:"confirmation_number"`
	Email     string            `json:"contact_email"`
	Comment   string            `json:"text_data"`
}

type filings struct {
	Entry []entry `json:"filings"`
}

func main() {
	fetchFilings()
	//analyze()
}

func fetchFilings(){
	var allFilings filings
	limit := 250
	offset := 37500
	for {
		url := "https://ecfsapi.fcc.gov/filings?limit="+strconv.Itoa(limit)+"&offset="+strconv.Itoa(offset)+"&proceedings.name=18-197&q=18%5C-197&sort=date_disseminated,DESC"

		//fmt.Println("Current offset: ", offset)
		//fmt.Println(url)
		response, err := http.Get(url)
		data, _ := ioutil.ReadAll(response.Body)

		if err != nil {
			fmt.Printf("The HTTP request failed with error %s\n", err)
			os.Exit(-1)
		}

		filings := load(data)

		fmt.Println("Got ", len(filings.Entry), " filings.")

		if &allFilings == nil {
			allFilings = filings
		} else {
			allFilings.Entry = append(allFilings.Entry, filings.Entry...)
		}

		if len(filings.Entry) == 0 {
			fmt.Println("Reached the end of the filings.")
			break;
		}

		offset += limit
		time.Sleep(500*time.Millisecond) //To not DOS the FCC
	}
	fmt.Println("Total filings grabbed: ", len(allFilings.Entry))
	dumpFilings(allFilings)
	fmt.Println("Done!")
}

func analyze(){
	// Instantiate the Watson Natural Language Understanding service
	service, serviceErr := nlu.
		NewNaturalLanguageUnderstandingV1(&nlu.NaturalLanguageUnderstandingV1Options{
			URL:      "YOUR SERVICE URL",
			Version:  "2017-02-27",
			Username: "YOUR SERVICE USERNAME",
			Password: "YOUR SERVICE PASSWORD",
		})

	// Check successful instantiation
	if serviceErr != nil {
		panic(serviceErr)
	}
}

func dumpFilings(filings filings) {
	newYaml, _ := yaml.Marshal(filings)
	newJson, _ := yaml.YAMLToJSON(newYaml)
	fmt.Println(string(newJson))

	f, err := os.Create("fetched.json")
	if err != nil {
		fmt.Printf("Couldn't write filings.", err)
		os.Exit(-1)
	}
	defer f.Close()

	bytesWrote, err := f.WriteString(string(newJson))
	if err != nil {
		fmt.Printf("Couldn't write filings.", err)
		os.Exit(-1)
	}
	fmt.Printf("wrote %d bytes\n", bytesWrote)

	return
}


func load(data []byte) filings {
	// Read in our YAML file.
	//data, err := ioutil.ReadFile("example.json")
	//if err != nil {
	//	fmt.Printf("err: %v\n", err)
	//	os.Exit(-1)
	//}

	// Unmarshal the YAML into a struct.
	var filings filings
	err := yaml.Unmarshal(data, &filings)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		os.Exit(-1)
	}

	return filings
}

