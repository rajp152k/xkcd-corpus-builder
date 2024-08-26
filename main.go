package main


func main() {
	// TODO add sensible command line args
	// to save, load, populate corpus, etc
	//
	// TODO add throttling capabilities to deal with
	// call rate limitations

	crp := Corpus{}
	crp.Populate(true, 5)
}
