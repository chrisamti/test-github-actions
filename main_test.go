package main

import (
	"log"
	"os"
	"testing"

	feed_cbdockertester "github.com/DND-IT/feed-cbdockertester"
)

var (
	cbUser          = "Administrator"
	cbPassword      = "TestPassword"
	err             error
	fCBDockerTester *feed_cbdockertester.FeedCBDockerTest
)

func TestMain(m *testing.M) {
	log.Println("Do stuff BEFORE the tests!")
	// start local docker couchbase server
	fCBDockerTester, err = feed_cbdockertester.New("./testData/CB/initCB.toml", cbUser, cbPassword)
	if err != nil {
		log.Fatal(err)
	}
	exitVal := m.Run()

	defer func() {
		log.Println("Do stuff AFTER the tests!")

		err = fCBDockerTester.Purge()
		if err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
		os.Exit(exitVal)
	}()

}

func Test_someMagickFunc(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "test feed-cbdockertester"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			someMagickFunc()
		})
	}
}
