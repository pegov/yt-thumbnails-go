package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "github.com/pegov/yt-thumbnails-go/api/thumbnail_v1"
)

var (
	addr                = flag.String("addr", "localhost:8080", "address")
	inputFile           = flag.String("input", "", "input file with newline separated urls")
	output              = flag.String("output", ".", "output path")
	async               = flag.Bool("async", false, "make requests in parallel")
	maxParallelRequests = flag.Int("max-parallel-requests", 8, "max parallel requests")
	maxRetries          = flag.Int("max-retries", 3, "max retries if service is unavailable (0 - no retries)")
)

var retryPolicyTemplate = `{
"methodConfig": [{
	"name": [{"service": "ThumbnailService"}],
	"waitForReady": true,
	"retryPolicy": {
		"MaxAttempts": %d,
		"InitialBackoff": ".01s",
		"MaxBackoff": ".01s",
		"BackoffMultiplier": 1.0,
		"RetryableStatusCodes": [ "UNAVAILABLE" ]
	}
}]}`

func main() {
	flag.Parse()

	log.SetFlags(0)

	// Create output folder if it does not exist
	if info, err := os.Stat(*output); !os.IsNotExist(err) {
		if !info.IsDir() {
			log.Fatalf("%v is not a dir!", output)
		}
	} else {
		if err := os.MkdirAll(*output, os.ModePerm); err != nil {
			log.Fatalf("Could not create folder with path %v\n", *output)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	retryPolicy := fmt.Sprintf(retryPolicyTemplate, *maxRetries)
	conn, err := grpc.Dial(
		*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(retryPolicy),
	)
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewThumbnailServiceClient(conn)

	var urls []string

	// Append urls from input file
	if *inputFile != "" {
		file, err := os.Open(*inputFile)
		if err != nil {
			log.Fatalf("Could not open %s: %v", *inputFile, err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if len(line) > 0 {
				urls = append(urls, scanner.Text())
			}
		}
	}

	// Append urls from args
	urls = append(urls, flag.Args()...)

	if len(urls) == 0 {
		log.Fatalf("No urls specified")
	}

	log.Printf("Total number of urls: %d", len(urls))

	if !*async {
		log.Printf("async: OFF")

		var successfullOps int
		for _, url := range urls {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			res, err := c.Get(ctx, &pb.GetRequest{Url: url})
			if err != nil {
				logError(url, err)
			} else {
				b := res.GetData()
				videoID := res.GetVideoId()
				fullPath := writeFile(videoID, b, *output)
				log.Printf("Saved %s\n", fullPath)
				successfullOps++
			}
		}

		log.Printf(
			"Downloaded %d/%d thumbnails",
			successfullOps,
			len(urls),
		)

		return
	}

	log.Printf("async: ON")
	log.Printf("Number of parallel requests: %d", *maxParallelRequests)

	var successfullOps atomic.Int64
	var wg sync.WaitGroup
	wg.Add(len(urls))

	semaphore := make(chan struct{}, *maxParallelRequests)
	for _, url := range urls {
		go func(url string) {
			semaphore <- struct{}{}
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			res, err := c.Get(ctx, &pb.GetRequest{Url: url})
			<-semaphore

			if err != nil {
				logError(url, err)
			} else {
				b := res.GetData()
				videoID := res.GetVideoId()
				fullPath := writeFile(videoID, b, *output)
				log.Printf("Saved %s\n", fullPath)
				successfullOps.Add(1)
			}
			wg.Done()
		}(url)
	}
	wg.Wait()

	log.Printf(
		"Downloaded %d/%d thumbnails",
		successfullOps.Load(),
		len(urls),
	)
}

func writeFile(videoID string, b []byte, outputPath string) string {
	const NewFileFormat = "%s.jpg"
	filename := fmt.Sprintf(NewFileFormat, videoID)
	p := path.Join(outputPath, filename)
	os.WriteFile(p, b, 0666)
	return p
}

func logError(url string, err error) {
	switch status.Code(err) {
	case codes.NotFound:
		log.Printf("%v: not found", url)
	case codes.InvalidArgument:
		log.Printf("%v: invalid url", url)
	case codes.DeadlineExceeded:
		log.Printf("%v: timeout", url)
	case codes.Canceled:
		log.Printf("%v: canceled context", url)
	case codes.Unavailable:
		log.Printf("%v: unavailable", url)
	default:
		log.Printf("%v: %v", url, err)
	}
}
