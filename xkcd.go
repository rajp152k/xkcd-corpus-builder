package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
)

type Service struct {
	XKCDEndpoint string
	JSONSuffix   string
}

type Comic struct {
	Num        int    `json:"num"`
	Day        string `json:"day"`
	Month      string `json:"month"`
	Year       string `json:"year"`
	Transcript string `json:"transcript"`
	Img        string `json:"img"`
	Title      string `json:"title"`
}

func (cmc Comic) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Struct Comic\n"))
	v := reflect.ValueOf(cmc)
	t := reflect.TypeOf(cmc)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		sb.WriteString(fmt.Sprintf("%s: %v\n", field.Name, value))
	}
	return sb.String()
}

type Corpus map[int]*Comic

var Srv *Service
var Lgr *log.Logger
var Crp Corpus

func init() {
	Srv = &Service{
		XKCDEndpoint: `https://xkcd.com/`,
		JSONSuffix:   `/info.0.json`,
	}
	Lgr = log.New(os.Stdout, "[XKCD] ", log.LstdFlags)
	Crp = Corpus{}
}

const (
	BoundFinderMax = 1e5
)

func BoundFinder(presentP func(int) bool) (int, int) {
	curr := 1
	for {
		Lgr.Println("curr: ", curr)
		if curr > BoundFinderMax {
			Lgr.Println("Bound Finder's Capabilities exceeded, terminating with sentinel returns")
			return 0, 0
		}
		if presentP(curr) {
			curr *= 2
		} else {
			return int(curr / 2), curr
		}
	}
}

func BinarySearchFlip(low int, high int, flipP func(int) bool) int {
	//Returns the largest index between low to high that triggers the flip Predicate
	if low > high {
		Lgr.Printf("Invalid request to BinarySearchFlip (low > high): %d, %d\n", low, high)
		return 0
	}
	for {
		Lgr.Println("low: ", low, "| high: ", high)
		if low == high {
			return high
		}
		mid := int((low + high) / 2)
		if flipP(mid) {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
}

func (crp Corpus) PresentP(id int) bool {
	resp, err := http.Get(fmt.Sprintf("%s%d%s", Srv.XKCDEndpoint, id, Srv.JSONSuffix))
	if err != nil {
		Lgr.Printf("Unable to Fetch Response")
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		Lgr.Printf("Received %d for comic %d\n", resp.StatusCode, id)
		return false
	}
	return true
}

func (crp Corpus) FetchComic(id int) (*Comic, error) {

	resp, err := http.Get(fmt.Sprintf("%s%d%s", Srv.XKCDEndpoint, id, Srv.JSONSuffix))
	if err != nil {
		Lgr.Printf("Unable to fetch comic %d\n", id)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		Lgr.Printf("Received %d for comic %d\n", resp.StatusCode, id)
		return nil, err
	}

	cmc := Comic{}

	err = json.NewDecoder(resp.Body).Decode(&cmc)

	if err != nil {
		Lgr.Printf("Unable to Unmarshal JSON object into comic for id: %d\n", id)
		return nil, err
	}

	return &cmc, nil
}

func (crp Corpus) MemberP(id int) bool {
	_, ok := crp[id]
	return ok
}

func (crp Corpus) Insert(id int, cmc *Comic) error {
	if crp.MemberP(id) {
		Lgr.Printf("id %d already present in corpus\n", id)
		return fmt.Errorf("id %d already present in corpus\n", id)
	}
	crp[id] = cmc
	return nil
}

func (crp Corpus) Delete(id int) error {
	if !crp.MemberP(id) {
		Lgr.Printf("invalid request: id %d not present in corpus\n", id)
		return fmt.Errorf("invalid request: id %d not present in corpus\n", id)
	}
	delete(crp, id)
	return nil
}

func (crp Corpus) SearchLimit() int {
	low, high := BoundFinder(crp.PresentP)
	return BinarySearchFlip(low, high, crp.PresentP)
}

func (crp Corpus) Populate(useTestLimit bool, numConns int) {
	defer crp.SaveToFile("./XKCDCorpus.json")
	Lgr.Println("Populating Corpus")
	Lgr.Println("Finding Comic Limit")
	var limit int
	if !useTestLimit {
		limit = crp.SearchLimit()
	} else {
		limit = 10
	}
	Lgr.Printf("Found Limit as %d\n", limit)

	var wg sync.WaitGroup
	wg.Add(limit)

	jobIds := make(chan int, limit)

	//populating Jobs Ids
	for i := 1; i <= limit; i++ {
		go func(id int) {
			defer wg.Done()
			jobIds <- i
		}(i)
		// go func(id int) {
		// 	defer wg.Done()
		// 	cmc, err := crp.FetchComic(id)
		// 	if err != nil {
		// 		Lgr.Printf("Error encountered when fetching %d: %v\n", id, err)
		// 		return
		// 	}
		// 	err = crp.Insert(id, cmc)
		// 	if err != nil {
		// 		Lgr.Printf("Error encountered when inserting %d: %v\n", id, err)
		// 		return
		// 	}
		// 	Lgr.Printf("Inserting Comic %d\n", id)
		// }(i)
	}
	wg.Wait()
	close(jobIds)

	//dispatching consumers
	wg.Add(numConns)

	for i := 0; i < numConns; i++ {
		go func(connNum int) {
			defer wg.Done()
			for job := range jobIds {
				Lgr.Printf("conn %d processing comic %d\n", connNum, job)
				//proceeding with mutex-free writes
				//as each job id is certainly different
				cmc, err := crp.FetchComic(job)
				if err != nil {
					Lgr.Printf("Error encountered when fetching %d: %v\n", job, err)
					return
				}
				err = crp.Insert(job, cmc)
				if err != nil {
					Lgr.Printf("Error encountered when inserting %d: %v\n", job, err)
					return
				}
				Lgr.Printf("Inserting Comic %d\n", job)
			}
		}(i)
	}

	wg.Wait()
	Lgr.Printf("Populated corpus with %d records\n", limit)
}

func (crp Corpus) Save(w io.Writer) error {
	encoding, err := json.Marshal(crp)
	if err != nil {
		Lgr.Printf("Error Marshalling Corpus: %v\n", err)
		return err
	}

	_, err = w.Write(encoding)
	if err != nil {
		Lgr.Printf("Error Writing Corpus into %v: %v\n", w, err)
		return err
	}
	return nil
}

func (crp Corpus) SaveToFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		Lgr.Printf("Error opening %s: %v", filename, err)
		return err
	}
	defer f.Close()

	return crp.Save(f)
}

func (crp Corpus) Load(r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		Lgr.Printf("Error Loading into Corpus: %v", err)
		return err
	}
	err = json.Unmarshal(data, &crp)
	if err != nil {
		Lgr.Printf("Error Loading into Corpus: %v", err)
		return err
	}
	return nil
}

func (crp Corpus) LoadFromFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		Lgr.Printf("Error opening %s: %v", filename, err)
		return err
	}
	defer f.Close()

	return crp.Load(f)
}
