package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
	"golift.io/xtractr"
)

var (
	file = flag.String("file", "", "file to crack")
)

func main() {
	flag.Parse()

	// Initialize debug logging
	devLog, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println(err)
		return
	}
	log := devLog.Sugar()
	defer log.Sync()

	// The alphabet to use
	alphabet := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890`-=[]\\;',./~!@#$%^&*()_+{}|:\"<>? ")
	// We'll generate passwords here
	var queue []string
	// TODO: improve this dummy algorithm for generating passwords of arbitrary length
	start := ""
	preStart := ""
	for a := 0; a < len(alphabet); a++ {
		for j := 0; j < len(alphabet); j++ {
			for i := 0; i < len(alphabet); i++ {
				queue = append(queue, start+string(alphabet[i]))
			}
			start = preStart + string(alphabet[j])
		}
		preStart = string(alphabet[a])
	}
	log.Infof("alphabet: %v", string(alphabet))
	log.Infof("generated %v words", len(queue))

	// Create the progress bar
	progressBar := pb.Full.New(len(queue))
	progressBar.SetRefreshRate(1 * time.Second)
	progressBar.Start()

	// We'll use a wait group to wait for all goroutines to finish
	var wg sync.WaitGroup
	startTime := time.Now()
	doneChan := make(chan bool, 1)

	// This code will seriously peg the CPU.
	// This is an attempt to release the CPU usage so the system remains somewhat useful.
	semCpus := semaphore.NewWeighted(int64(runtime.NumCPU() - 1))
	ctx := context.Background()
main_loop:
	for index, pass := range queue {
		select {
		// Did we discover the password?
		case <-doneChan:
			break main_loop
		default:
		}

		semCpus.Acquire(ctx, 1)
		wg.Add(1)
		go func(index int, pass string) {
			defer wg.Done()
			// We won't run more than runtime.NumCPU() - 1 goroutines
			defer semCpus.Release(1)

			defer func() {
				if err := recover(); err != nil {
					log.Errorf("panic: %v: %s", err, debug.Stack())
				}
			}()

			x := &xtractr.XFile{
				FilePath: *file,
				// Dummy directory to try to extract the archive to.
				// TODO: automatically remove the directory when done
				OutputDir: *file + "-output",
				Passwords: []string{pass},
				DirMode:   os.FileMode(0750),
				FileMode:  os.FileMode(0750),
			}

			// size is how many bytes were written.
			// files may be nil, but will contain any files written (even with an error).
			size, _, _, err := xtractr.ExtractFile(x)
			if err == nil && size > 0 {
				log.Infof("Password found: '%v'. Success.", pass)
				doneChan <- true
			}
			progressBar.Increment()
		}(index, pass)
	}
	wg.Wait()
	log.Infof("duration %v", time.Since(startTime))
}
