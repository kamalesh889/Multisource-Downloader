package service

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

const (
	chunkSize     = 1024 * 1024
	maxGoroutines = 200
)

// var (
// 	fileUrl = "https://cdn.videvo.net/videvo_files/video/premium/video0042/large_watermarked/900-2_900-6334-PD2_preview.mp4"
// )

// var (
// 	fileUrl = "https://getsamplefiles.com/download/mp4/sample-1.mp4"
// )

var (
	fileUrl = "https://sample-videos.com/video123/mp4/480/big_buck_bunny_480p_30mb.mp4"
)

// var (
// 	fileUrl = "https://filesamples.com/samples/video/mp4/sample_3840x2160.mp4" // Does not support partial content download
// )

type DownLoader struct {
	url      string
	store    map[int64][]byte
	maxChunk int64
	client   *http.Client
	wg       *sync.WaitGroup
	mu       *sync.Mutex
}

func NewDownLoader() *DownLoader {
	d := &DownLoader{}

	client := &http.Client{}
	storeMap := make(map[int64][]byte)

	d.store = storeMap
	d.client = client
	d.url = fileUrl

	d.wg = new(sync.WaitGroup)
	d.mu = new(sync.Mutex)

	return d
}

func (d *DownLoader) DownloadFile() {

	isDownload := d.IsDownloadable()
	if !isDownload {
		fmt.Println("Can not download the file with this Downloader")
		return
	}

	var idx int64
	exitSignal := false

	semaphore := make(chan struct{}, maxGoroutines)

	for !exitSignal {

		d.wg.Add(1)
		semaphore <- struct{}{}
		go d.DownloadInchunks(semaphore, chunkSize*idx, &exitSignal)

		d.mu.Lock()
		idx++
		d.mu.Unlock()

	}

	d.wg.Wait()

	file, err := os.Create("file.mp4")
	if err != nil {
		fmt.Println("error in creating the file", err)
		return
	}

	var chunk int64

	for chunk <= d.maxChunk {
		_, err := file.Write(d.store[chunk])
		if err != nil {
			fmt.Println("error in assamble", err)
			return
		}
		chunk = chunk + chunkSize
	}

	defer file.Close()

}

func (d *DownLoader) DownloadInchunks(semaphore chan struct{}, pointerChunk int64, exitSig *bool) {

	defer func() {
		<-semaphore
		d.wg.Done()
	}()

	req, err := http.NewRequest("GET", d.url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", pointerChunk, pointerChunk+chunkSize-1))

	resp, err := d.client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	d.mu.Lock()
	d.store[pointerChunk] = data
	if d.maxChunk < pointerChunk {
		d.maxChunk = pointerChunk
	}
	d.mu.Unlock()

	if resp.StatusCode != http.StatusPartialContent {
		*exitSig = true
	}

}

// To check Server supports partial content download or not
func (d *DownLoader) IsDownloadable() bool {

	req, err := http.NewRequest("GET", d.url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return false
	}

	req.Header.Set("Range", "bytes=0-499")

	resp, err := d.client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusPartialContent {
		return true
	} else {
		return false
	}

}
