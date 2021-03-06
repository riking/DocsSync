package main

import (
	"encoding/json"
	"fmt"
	"github.com/skratchdot/open-golang/open"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	drive "google.golang.org/api/drive/v2"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"sync"
	"syscall"
	"time"
)

type SyncConfig struct {
	Directory    string      `json:"directory"`
	ClientID     string      `json:"client_id"`
	ClientSecret string      `json:"client_secret"`
	Files        []SyncEntry `json:"files"`
}
type SyncEntry struct {
	Filename string `json:"filename"`
	FileId   string `json:"file_id"`
	Mime     string `json:"mime"`
}

type MyTokenSource struct {
	context context.Context
}

type SkippedDownload struct{}

var config = &oauth2.Config{
	ClientID:     "",
	ClientSecret: "",
	Scopes:       []string{drive.DriveReadonlyScope},
	RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://accounts.google.com/o/oauth2/auth",
		TokenURL: "https://accounts.google.com/o/oauth2/token",
	},
}

var _ = &http.Transport{}
var _ = reflect.TypeOf

var syncConf SyncConfig

func (s SkippedDownload) Error() string {
	return "Download skipped"
}

func readConfig() error {
	confFile, err := os.Open("sync_config.json")
	if err != nil {
		return err
	}
	defer confFile.Close()
	dec := json.NewDecoder(confFile)
	err = dec.Decode(&syncConf)
	return err
}

func authorize(ctx context.Context) (token *oauth2.Token, err error) {
	var err2 error
	// Read previous authorization
	func() {
		tokenFile, err2 := os.OpenFile(".rftoken", os.O_RDONLY, 0600)
		if err2 != nil {
			return
		}
		defer tokenFile.Close()
		dec := json.NewDecoder(tokenFile)
		err2 = dec.Decode(&token)
	}()

	if err2 != nil {
		// Check the error
		patherr := err2.(*os.PathError)
		errno := patherr.Err.(*syscall.Errno)
		if *errno == syscall.EPERM {
			log.Printf("Permission denied trying to open .rftoken\n")
			return nil, patherr
		} else if *errno == syscall.ENOENT {
			// ENOENT is OK
			// mark as expired
			token.Expiry = time.Unix(10, 0)
		} else {
			log.Fatalf("Error trying to open .rftoken: %v\n", patherr)
			return nil, patherr
		}
	}

	mysource := MyTokenSource{ctx}
	tsource := oauth2.ReuseTokenSource(token, &mysource)

	token, err = tsource.Token()
	return
}

func (s *MyTokenSource) Token() (token *oauth2.Token, err error) {
	authUrl := config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	log.Printf("Go to the following link in your browser: %v\n", authUrl)
	open.Start(authUrl)

	// Read the code, and exchange it for a token.
	fmt.Printf("\n\nEnter verification code: ")
	var code string
	fmt.Scanln(&code)
	fmt.Println()
	token, err = config.Exchange(s.context, code)
	if err != nil {
		return nil, err
	}

	var err2 error
	// Write out refresh token
	saveFile, err2 := os.OpenFile(".rftoken", os.O_WRONLY|os.O_CREATE, 0600)
	if err2 != nil {
		log.Printf("Error saving access token: %v\n", err2)
	}
	defer saveFile.Close()
	enc := json.NewEncoder(saveFile)
	err2 = enc.Encode(token)

	if err2 != nil {
		log.Printf("Error saving access token: %v\n", err2)
	}

	return token, nil
}

func downloadFile(
	entry SyncEntry,
	fileSvc *drive.FilesService,
	httpClient *http.Client) error {

	// Get the Drive file information
	fileRequest := fileSvc.Get(entry.FileId)
	dFile, err := fileRequest.Do()
	if err != nil {
		log.Printf("File %s - Drive error: %v\n", entry.Filename, err)
		return err
	}

	// Open the destination file
	outFile, err := os.OpenFile(entry.Filename, os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("File %s - Could not open: %v\n", entry.Filename, err)
		return err
	}
	defer outFile.Close()

	// Check the remote vs local modification times
	remoteModtime, err := time.Parse(time.RFC3339, dFile.ModifiedDate)
	if err != nil {
		log.Panicf("Could not parse time in response - file %s string %s\n", entry.Filename, dFile.ModifiedDate)
	}
	outInfo, err := outFile.Stat()
	if err != nil {
		log.Printf("File %s - Could not stat: %v\n", entry.Filename, err)
		return err
	}
	localModtime := outInfo.ModTime()

	if remoteModtime.Before(localModtime) {
		log.Printf("File %s is up to date\n", entry.Filename)
		return &SkippedDownload{}
	}

	// Set up the download
	download_url := dFile.ExportLinks[entry.Mime]
	log.Printf("Downloading %s from %s\n", entry.Filename, entry.FileId)

	resp, err := httpClient.Get(download_url)
	if err != nil {
		log.Printf("File %s - Download failed: %v\n", entry.Filename, err)
		return err
	}
	defer resp.Body.Close()

	// Finish the download
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		log.Printf("File %s - write error: %v\n", entry.Filename, err)
		return err
	}
	return nil
}

func main() {
	// Read the config, move to that directory
	err := readConfig()
	if err != nil {
		log.Fatalf("An error occured reading the config file: %v\n", err)
	}
	os.Chdir(syncConf.Directory)

	config.ClientID = syncConf.ClientID
	config.ClientSecret = syncConf.ClientSecret

	// Get authorization
	ctx := context.TODO()
	token, err := authorize(ctx)
	if err != nil {
		log.Fatalf("An error occured with authorization: %v\n", err)
	}

	// Create a new Drive client object.
	svc, err := drive.New(config.Client(ctx, token))
	if err != nil {
		log.Fatalf("An error occurred creating Drive client: %v\n", err)
	}

	fileSvc := drive.NewFilesService(svc)
	httpClient := config.Client(ctx, token)
	wg := &sync.WaitGroup{}

	// Download the files
	for _, entry := range syncConf.Files {
		go func(entry SyncEntry) {
			err2 := downloadFile(entry, fileSvc, httpClient)
			if err2 != nil {
				if _, ok := err2.(*SkippedDownload); ok {
					log.Printf("IGN %s", entry.Filename)
				} else {
					log.Printf("ERR %s", entry.Filename)
				}
			} else {
				log.Printf("OK  %s", entry.Filename)
			}
			wg.Done()
		}(entry)
		wg.Add(1)
	}

	wg.Wait()
}
