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
)

func WriteFile(videoID string, b []byte, outputPath string) string {
	const NewFileFormat = "%s.jpg"
	filename := fmt.Sprintf(NewFileFormat, videoID)
	p := path.Join(outputPath, filename)
	os.WriteFile(p, b, 0666)
	return p
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	// Create output folder if not exists
	if info, err := os.Stat(*output); !os.IsNotExist(err) {
		if !info.IsDir() {
			log.Fatalf("%v is not a dir!", output)
		}
	} else {
		if err := os.MkdirAll(*output, os.ModePerm); err != nil {
			log.Fatalf("Could not create folder with path %v\n", *output)
		}
	}

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewThumbnailServiceClient(conn)

	ctx := context.Background()

	var urls []string

	// Either input file OR args
	if *inputFile != "" {
		file, err := os.Open(*inputFile)
		if err != nil {
			log.Fatalf("Could not open %s: %v", *inputFile, err)
		}
		defer file.Close()

		var lines []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if len(line) > 0 {
				lines = append(lines, scanner.Text())
			}
		}
		urls = lines
	} else {
		urls = flag.Args()
	}

	log.Printf("Total number of urls: %d", len(urls))

	if len(urls) == 0 {
		log.Fatalf("No urls specified")
	}

	if *async {
		var successfullOps atomic.Int64
		log.Printf("async: ON")
		log.Printf("Number of parallel requests: %d", *maxParallelRequests)
		var wg sync.WaitGroup
		wg.Add(len(urls))
		semaphore := make(chan struct{}, *maxParallelRequests)
		for _, url := range urls {
			go func(url string) {
				semaphore <- struct{}{}
				ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
				r, err := c.Get(ctx, &pb.GetRequest{Url: url})
				<-semaphore
				if err != nil {
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
						cancel()
					default:
						log.Printf("%v: %v", url, err)
					}
				} else {
					b := r.GetData()
					name := r.GetVideoId()
					fullPath := WriteFile(name, b, *output)
					log.Printf("Saved %s\n", fullPath)
					successfullOps.Add(1)
				}
				wg.Done()
			}(url)
		}
		wg.Wait()

		err := ctx.Err()
		if err != nil {
			log.Fatalf("Could not connect to server")
		}

		log.Printf(
			"Downloaded %d/%d thumbnails",
			successfullOps.Load(),
			len(urls),
		)
	} else {
		log.Println("async: OFF")
		successfullOps := 0
		for _, url := range urls {
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			r, err := c.Get(ctx, &pb.GetRequest{Url: url})
			if err != nil {
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
					cancel()
				default:
					log.Printf("Could not get %v: %v", url, err)
				}
			} else {
				b := r.GetData()
				name := r.GetVideoId()
				fullPath := WriteFile(name, b, *output)
				log.Printf("Saved %s\n", fullPath)
				successfullOps++
			}
		}

		err := ctx.Err()
		if err != nil {
			log.Fatalf("Could not connect to server")
		}

		log.Printf(
			"Downloaded %d/%d thumbnails",
			successfullOps,
			len(urls),
		)
	}

}
