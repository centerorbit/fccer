package main

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/joho/godotenv"
	nlu "github.com/watson-developer-cloud/go-sdk/naturallanguageunderstandingv1"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
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

type intersection struct {
	Count int
	Comment string
}

func main() {
	var filings filings
	there, _ := exists("fetched.json")

	if there {
		filings = loadFromFile("fetched.json")
	} else {
		filings = fetchFilings()
	}
	fmt.Println("Loaded ", len(filings.Entry), " filings.")

	intersections := filter(filings) // need to return the uniques, to use for the analyze
	analyze(intersections)
}

func filter(filings filings) []intersection {

	//First, lets sort the filings
	sort.Slice(filings.Entry, func(i, j int) bool {
		return filings.Entry[i].Comment < filings.Entry[j].Comment
	})

	var intersections []intersection
	var firstOccurrence *string
	var foundIntersection *intersection
	totalDuplicates := 0
	for i:= 0; i < len(filings.Entry); i ++ {
		if firstOccurrence == nil {
			firstOccurrence = &filings.Entry[i].Comment
			multiples := intersection{Comment: *firstOccurrence, Count: 1}
			foundIntersection = &multiples
			intersections = append(intersections, multiples)
		} else if foundIntersection != nil && *firstOccurrence == filings.Entry[i].Comment {
			foundIntersection.Count++
			totalDuplicates++
		} else {
			//if foundIntersection.Count > 1 {
			//	hash := md5.Sum([]byte(foundIntersection.Comment))
			//	fmt.Println("No more duplicates for ",hex.EncodeToString(hash[:]), " with ", foundIntersection.Count, " copies found.")
			//}

			i --
			foundIntersection = nil
			firstOccurrence = nil
		}
	}

	fmt.Println("Filtered down to a total of:", len(intersections), " uniques.")
	fmt.Println("With a total of: ", totalDuplicates, " duplicates removed.")

	return intersections
}


func fetchFilings() filings {
	var allFilings filings
	limit := 250
	offset := 0
	proceeding := "18-197"

	for {
		url := "https://ecfsapi.fcc.gov/filings?limit="+strconv.Itoa(limit)+"&offset="+strconv.Itoa(offset)+"&proceedings.name="+proceeding+"&sort=date_disseminated,DESC"

		fmt.Println("Requesting at offset: ", offset)
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

func analyze(filings []intersection){
	_ = godotenv.Load("ibm-credentials.env")

	// Instantiate the Watson Natural Language Understanding service
	service, serviceErr := nlu.
		NewNaturalLanguageUnderstandingV1(&nlu.NaturalLanguageUnderstandingV1Options{
			Version:  "2017-02-27",
			URL:       os.Getenv("NATURAL_LANGUAGE_UNDERSTANDING_URL"),
			IAMApiKey: os.Getenv("NATURAL_LANGUAGE_UNDERSTANDING_APIKEY"),
		})

	// Check successful instantiation
	if serviceErr != nil {
		panic(serviceErr)
	}

	analyzeLimit := 10000
	emptyComments := 0
	inFavor := 0
	inOpposition := 0
	var sentimentTotal float64 = 0
	i:= 0
	for ; i <= analyzeLimit && i < len(filings); i ++ {
		fmt.Println("Analyzing ",i)
		entry := filings[i]

		if entry.Comment == "" { //Can't analyze nothing.
			emptyComments++
			continue
		}

		analyzeOptions := service.NewAnalyzeOptions(&nlu.Features{
			Sentiment: &nlu.SentimentOptions{},
			//Entities: &nlu.EntitiesOptions{},
			//Keywords: &nlu.KeywordsOptions{},
		}).SetText(entry.Comment)

		// Call the naturalLanguageUnderstanding Analyze method
		response, responseErr := service.Analyze(analyzeOptions)

		// Check successful call
		if responseErr != nil {
			fmt.Println("Couldn't process ", entry.Comment)
			fmt.Println(responseErr)
			continue
		}

		// Print the entire detailed response
		//fmt.Println(response)

		// Cast analyze.Result to the specific dataType returned by Analyze
		// NOTE: most methods have a corresponding Get<methodName>Result() function
		analyzeResult := service.GetAnalyzeResult(response)

		// Check successful casting
		if analyzeResult != nil {
			score := *analyzeResult.Sentiment.Document.Score

			if score > 0 {
				inFavor ++
			} else {
				inOpposition ++
			}

			sentimentTotal += score
		}
	}

	averageSentiment := (sentimentTotal/float64(i))
	fmt.Println("A total of ", i, " filings were processed." )
	fmt.Println("The average sentiment of these filings is ", averageSentiment)
	fmt.Println("Positive one means in approval, Negative one means in opposition.")
	fmt.Print("Generally speaking, the comments are ")

	if averageSentiment > 0 {
		fmt.Println("in favor of the merger.")
	} else {
		fmt.Println("in opposition of the merger.")
	}

	fmt.Println("There were ", inFavor, " people that were in favor of the merger, and ", inOpposition, " people in opposition.")
}

func printFilings(filings filings)[]byte{
	newYaml, _ := yaml.Marshal(filings)
	newJson, _ := yaml.YAMLToJSON(newYaml)
	fmt.Println(string(newJson))
	return newJson
}


func dumpFilings(filings filings) {
	newJson := printFilings(filings)

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