package main

import (
	"encoding/json"
	"testing"
)

func TestCorpusOps(t *testing.T) {
	test_crp := Corpus{}
	Equal(t, test_crp.MemberP(0), false)


	test_crp.Insert(1, &Comic{})
	Equal(t, test_crp.MemberP(1), true)

	Equal(t, test_crp.Delete(1), nil)

	test_crp.Delete(0)
	Equal(t, test_crp.Delete(0).Error(), "invalid request: id 0 not present in corpus\n")

	Equal(t, test_crp.PresentP(0), false)
	Equal(t, test_crp.PresentP(1), true)
	Equal(t, test_crp.PresentP(10000000), false)
	Equal(t, test_crp.PresentP(-1), false)
}

func TestComicFetch(t *testing.T) {

	test_crp := Corpus{}

	comic, _ := test_crp.FetchComic(0)
	Equal(t, comic, nil)

	fetch, _ := test_crp.FetchComic(1)

	expected_json := []byte(`{
  "month": "1",
  "num": 1,
  "link": "",
  "year": "2006",
  "news": "",
  "safe_title": "Barrel - Part 1",
  "transcript": "[[A boy sits in a barrel which is floating in an ocean.]]\nBoy: I wonder where I'll float next?\n[[The barrel drifts into the distance. Nothing else can be seen.]]\n{{Alt: Don't we all.}}",
  "alt": "Don't we all.",
  "img": "https://imgs.xkcd.com/comics/barrel_cropped_(1).jpg",
  "title": "Barrel - Part 1",
  "day": "1"
}`)

	expected_unmarshal := Comic{}

	json.Unmarshal(expected_json, &expected_unmarshal)
	Equal(t, *fetch, expected_unmarshal)
}

func TestSearchLimit(t *testing.T){
	test_crp := Corpus{}

	Lgr.Println(test_crp.SearchLimit())
}

func TestPopulate(t *testing.T){
	test_crp := Corpus{}

	test_crp.Populate(true, 2)

	for i,cmc := range test_crp {
		Lgr.Println("Comic: ",i,*cmc)
	}
}

func TestSave(t *testing.T) {
	t.Helper()
	test_crp := Corpus{}

	test_crp.Populate(true,2)

	Equal(t, test_crp.SaveToFile("./testSave.json"), nil)
}

func TestLoad(t *testing.T) {
	t.Helper()
	test_crp := Corpus{}

	Equal(t, test_crp.LoadFromFile("./testSave.json"),nil)
	Lgr.Println(test_crp)
}
