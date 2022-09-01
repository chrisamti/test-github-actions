package feed_cbdockertester

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/couchbase/gocb/v2"
	"github.com/ory/dockertest/v3"
)

var ErrTOML = errors.New("error reading toml file")

// InitData defines method, path, data to be send on initialization of the test couchbase server
type InitData struct {
	Method   string
	Path     string
	Data     string
	Datapath string
	Auth     bool
	Info     string
	Port     string
}

// InitDataItems implements array of InitData
type InitDataItems struct {
	InitData []InitData `toml:"init"`
}

type credentials struct {
	User     string
	Password string
}

type TestData struct {
	Key  string
	Data interface{}
}

type TestDataItems []TestData

type FeedCBDockerTest struct {
	pool         *dockertest.Pool
	resource     *dockertest.Resource
	pathInitData string
	credentials  credentials
}

// New creates a new couchbase docker based test server
func New(pathInitData string, user string, password string) (*FeedCBDockerTest, error) {
	var err error

	var fCBDTest = &FeedCBDockerTest{
		pathInitData: pathInitData,
		credentials: struct {
			User     string
			Password string
		}{
			User:     user,
			Password: password,
		},
	}

	fCBDTest.pool, err = dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	log.Print("starting testing couchbase env")
	fCBDTest.resource, err = fCBDTest.pool.Run("couchbase", "enterprise-7.1.1", nil)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	if fCBDTest.resource == nil {
		log.Fatalf("docker resource is nil")
	}

	// let docker kill container after 300 seconds ...
	err = fCBDTest.resource.Expire(300)
	if err != nil {
		log.Print(err)
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err = fCBDTest.pool.Retry(func() error {
		return fCBDTest.initCB(fCBDTest.resource.GetPort("8091/tcp"))
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
		fCBDTest.Purge()
		return fCBDTest, err
	}

	return fCBDTest, err
}

// GetRestPort wrapper for resource.GetRestPort
func (fCBDTest *FeedCBDockerTest) GetRestPort() string {
	return fCBDTest.resource.GetPort("8091/tcp")
}

// GetMemcachedPort wrapper for resource.GetRestPort
func (fCBDTest *FeedCBDockerTest) GetMemcachedPort() string {
	return fCBDTest.resource.GetPort("11210/tcp")
}

// GetN1QLPort wrapper for resource.GetRestPort
func (fCBDTest *FeedCBDockerTest) GetN1QLPort() string {
	return fCBDTest.resource.GetPort("8093/tcp")
}

// GetCapiPort wrapper for resource.GetRestPort
func (fCBDTest *FeedCBDockerTest) GetCapiPort() string {
	return fCBDTest.resource.GetPort("8092/tcp")
}

// GetFTSPort wrapper for resource.GetRestPort
func (fCBDTest *FeedCBDockerTest) GetFTSPort() string {
	return fCBDTest.resource.GetPort("8094/tcp")
}

// GetFTSGPRCPort wrapper for resource.GetRestPort
func (fCBDTest *FeedCBDockerTest) GetFTSGPRCPort() string {
	return fCBDTest.resource.GetPort("9130/tcp")
}

// Purge couchbase docker container
func (fCBDTest *FeedCBDockerTest) Purge() error {
	return fCBDTest.pool.Purge(fCBDTest.resource)
}

// initCB initializes the docker couchbase instance
func (fCBDTest *FeedCBDockerTest) initCB(port string) error {
	var (
		cbInit InitDataItems
		err    error
	)
	cbInit, err = fCBDTest.readCBInit()
	if err != nil {
		return err
	}

	// add alternative port setup
	// --> https://docs.couchbase.com/server/6.5/rest-api/rest-set-up-alternate-address.html
	var ap InitData
	ap.Info = "set dynamic docker ports"
	ap.Auth = true
	ap.Method = "PUT"
	ap.Path = "/node/controller/setupAlternateAddresses/external"
	ap.Data = fmt.Sprintf("hostname=localhost&mgmt=%s&kv=%s&n1ql=%s&capi=%s&fts=%s",
		fCBDTest.GetRestPort(),
		fCBDTest.GetMemcachedPort(),
		fCBDTest.GetN1QLPort(),
		fCBDTest.GetCapiPort(),
		fCBDTest.GetFTSPort(),
	)
	cbInit.InitData = append(cbInit.InitData, ap)

	for _, d := range cbInit.InitData {
		usePort := port

		switch d.Port {
		case "fts":
			usePort = fCBDTest.GetFTSPort()
		case "n1ql":
			usePort = fCBDTest.GetN1QLPort()
		}

		err = fCBDTest.doLazyWebCall(d, usePort)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fCBDTest *FeedCBDockerTest) readCBInit() (InitDataItems, error) {
	var (
		cbInitData InitDataItems
		err        error
	)

	// err = json.Unmarshal(buffer, &cbInitData)
	_, err = toml.DecodeFile(fCBDTest.pathInitData, &cbInitData)
	if err != nil {
		fmt.Println(err)
		return cbInitData, fmt.Errorf("%v: %w", ErrTOML, err)
	}

	return cbInitData, err
}

// GetTestData read data from json files and return as TestDataItems
func GetTestData(testFiles []string) (TestDataItems, error) {
	var err error
	var tds TestDataItems

	for _, fp := range testFiles {
		var td TestData

		td, err = readJsonFile(fp)

		if err != nil {
			log.Fatal(err)
		}

		tds = append(tds, td)

	}

	return tds, err
}

func readJsonFile(fp string) (TestData, error) {
	var f *os.File
	var err error

	f, err = os.Open(fp)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		err = f.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	byteValue, _ := ioutil.ReadAll(f)

	var td TestData
	err = json.Unmarshal(byteValue, &td.Data)
	if err != nil {
		log.Fatal(err)
	}

	td.Key = fileNameWithoutExtension(f.Name())

	return td, err
}

// AddTestData adds test data into couchbase bucket.
func (fCBDTest *FeedCBDockerTest) AddTestData(serverAddress string, bucketName string, data TestDataItems) (err error) {

	var (
		numBatches = 8 // number of batches
	)

	batches := make(map[int][]gocb.BulkOp)

	var cluster *gocb.Cluster
	cluster, err = gocb.Connect(serverAddress, gocb.ClusterOptions{
		Username: fCBDTest.credentials.User,
		Password: fCBDTest.credentials.Password,
	})

	if err != nil {
		return
	}

	defer func() {
		err = cluster.Close(nil)
		if err != nil {
			log.Print(err)
		}
	}()

	var bucket *gocb.Bucket
	var collection *gocb.Collection
	bucket = cluster.Bucket(bucketName)
	err = bucket.WaitUntilReady(40*time.Second, nil)
	if err != nil {
		log.Println(err)
		return
	}

	collection = bucket.DefaultCollection()

	if bucket != nil {
		for i, d := range data {
			batchNum := i % numBatches
			_, ok := batches[batchNum]
			if !ok {
				batches[batchNum] = []gocb.BulkOp{}
			}

			batches[batchNum] = append(batches[batchNum], &gocb.UpsertOp{
				ID:    d.Key,
				Value: d.Data,
			})
		}
	}

	for _, batch := range batches {
		err := collection.Do(batch, nil)
		if err != nil {
			log.Println(err)
		}

		for _, op := range batch {
			upsertOp := op.(*gocb.UpsertOp)
			if upsertOp.Err != nil {
				log.Println(upsertOp.Err)
			}

			_, errIsStored := ensureTestDataIsStored(collection, upsertOp.ID)
			if errIsStored != nil {
				log.Println(errIsStored)
			}
		}
	}

	return
}

// UploadTestData uploads all test data from filepath and add them into couchbase bucket.
func (fCBDTest *FeedCBDockerTest) UploadTestData(serverAddress, bucketName, filePath string) (err error) {
	var (
		testFiles []string
		tds       TestDataItems
	)

	testFiles, err = filepath.Glob(filePath)
	if err != nil {
		return
	}

	tds, err = GetTestData(testFiles)
	if err != nil {
		return
	}

	err = fCBDTest.AddTestData(serverAddress, bucketName, tds)
	if err != nil {
		return
	}

	return
}

func ensureTestDataIsStored(collection *gocb.Collection, key string) (isStored bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		return false, fmt.Errorf("could not get key %s", key)
	default:
		_, err = collection.Get(key, nil)
		if err == nil {
			isStored = true
			return
		}
	}

	return
}

func fileNameWithoutExtension(fileName string) string {
	return filepath.Base(strings.TrimSuffix(fileName, filepath.Ext(fileName)))
}
