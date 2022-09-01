package feed_cbdockertester

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"
)

func (fCBDTest *FeedCBDockerTest) doLazyWebCall(d InitData, port string) error {
	var (
		retries    = 10
		err        error
		statusCode int
	)

	for i := 1; i <= retries; i++ {
		statusCode, err = fCBDTest.doWebCall(d, port)
		if err == nil && statusCode < 300 && statusCode > 199 {
			return nil
		}

		log.Printf("try %d: couchbase server is not ready yet - waiting another 5 seconds", i)
		time.Sleep(time.Second * 5)
	}
	return err
}

func (fCBDTest *FeedCBDockerTest) doWebCall(d InitData, port string) (int, error) {
	var postValues url.Values
	var req *http.Request
	var resp *http.Response
	var err error
	var tmpl *template.Template
	var body *strings.Reader

	if d.Port == "fts" {
		bodyByte, err := ioutil.ReadFile(d.Datapath)
		if err != nil {
			log.Fatal(err)
		}

		body = strings.NewReader(string(bodyByte))
	} else {
		tmpl, err = template.New("").Parse(d.Data)
		if err != nil {
			log.Fatal(err)
		}

		var out bytes.Buffer

		if err := tmpl.Execute(&out, fCBDTest.credentials); err != nil {
			return 0, err
		}

		postValues, err = url.ParseQuery(out.String())
		if err != nil {
			log.Print(err)
		}

		body = strings.NewReader(postValues.Encode())
	}

	uri := fmt.Sprintf("http://localhost:%s%s", port, d.Path)
	req, err = http.NewRequest(d.Method, os.ExpandEnv(uri), body)
	if err != nil {
		log.Fatal(err)
	}

	if d.Port == "fts" {
		req.Header.Set("Content-Type", "application/json")
	} else {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	req.Header.Add("Cache-Control", "no-cache")

	if d.Auth {
		req.SetBasicAuth(fCBDTest.credentials.User, fCBDTest.credentials.Password)
	}

	client := &http.Client{}

	log.Printf("%s: calling %s", d.Info, uri)
	resp, err = client.Do(req)
	if err != nil {
		log.Print(err)
		return 0, err
	}

	var respBody []byte
	respBody, err = ioutil.ReadAll(resp.Body)

	defer func() {
		_ = resp.Body.Close()
	}()

	if err != nil {
		log.Fatal(err)
		return 0, err
	}

	log.Printf("status %d", resp.StatusCode)

	var b = string(respBody)
	if b != "" && b != "\"\"" {
		log.Printf("resp: '%s'", b)
	}

	return resp.StatusCode, nil
}
