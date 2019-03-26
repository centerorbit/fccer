package main

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/watson-developer-cloud/go-sdk/core"
	nlu "github.com/watson-developer-cloud/go-sdk/naturallanguageunderstandingv1"
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
	var filings filings
	there, _ := exists("fetched.json")

	if there {
		filings = loadFromFile("fetched.json")
	} else {
		filings = fetchFilings()
	}

	analyze(filings)
}

func fetchFilings() filings {
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

		filings := loadFromBytes(data)

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

	return allFilings
}

func analyze(filings filings){
	// Instantiate the Watson Natural Language Understanding service
	service, serviceErr := nlu.
		NewNaturalLanguageUnderstandingV1(&nlu.NaturalLanguageUnderstandingV1Options{
			Version:  "2017-02-27",
			URL:       "https://gateway.watsonplatform.net/natural-language-understanding/api",
			IAMApiKey: "api key here",
		})

	// Check successful instantiation
	if serviceErr != nil {
		panic(serviceErr)
	}

	entry := filings.Entry[1]

	analyzeOptions := service.NewAnalyzeOptions(&nlu.Features{
		Sentiment: &nlu.SentimentOptions{},
		//Entities: &nlu.EntitiesOptions{},
		//Keywords: &nlu.KeywordsOptions{},
	}).SetText(entry.Comment)

	// Call the naturalLanguageUnderstanding Analyze method
	response, responseErr := service.Analyze(analyzeOptions)

	// Check successful call
	if responseErr != nil {
		panic(responseErr)
	}

	// Print the entire detailed response
	fmt.Println(response)

	// Cast analyze.Result to the specific dataType returned by Analyze
	// NOTE: most methods have a corresponding Get<methodName>Result() function
	analyzeResult := service.GetAnalyzeResult(response)

	// Check successful casting
	if analyzeResult != nil {
		core.PrettyPrint(analyzeResult, "Analyze")
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


func loadFromFile(filename string) filings {
	// Read in our YAML file.
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		os.Exit(-1)
	}

	return loadFromBytes(data)
}

func loadFromBytes( data []byte) filings{
	// Unmarshal the YAML into a struct.
	var filings filings
	err := yaml.Unmarshal(data, &filings)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		os.Exit(-1)
	}

	return filings
}

// exists returns whether the given file or directory exists or not
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}