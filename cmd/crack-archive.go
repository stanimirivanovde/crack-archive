package main

import (
	"flag"
	"os"
	"runtime"
	"context"
	"runtime/debug"
	"sync"
	"time"
	"fmt"

	"github.com/cheggaaa/pb/v3"
	"golift.io/xtractr"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
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

	alphabet := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890`-=[]\\;',./~!@#$%^&*()_+{}|:\"<>? ")
	var queue []string
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

	progressBar := pb.Full.New(len(queue))
	progressBar.SetRefreshRate(1 * time.Second)
	progressBar.Start()
	var wg sync.WaitGroup
	startTime := time.Now()
	doneChan := make(chan bool, 1)
	semCpus := semaphore.NewWeighted(int64(runtime.NumCPU() - 1))
	ctx := context.Background()
main_loop:
	for index, pass := range queue {
		select {
		case <-doneChan:
			break main_loop
		default:
		}

		semCpus.Acquire(ctx, 1)
		wg.Add(1)
		go func(index int, pass string) {
			defer wg.Done()
			defer semCpus.Release(1)

			defer func() {
				// recover from panic if one occured. Set err to nil otherwise.
				// If we don't recover from a panic in a goroutine the entire app will crash.
				// This also includes the main thread.
				if err := recover(); err != nil {
					log.Errorf("panic: %v: %s", err, debug.Stack())
				}
			}()

			x := &xtractr.XFile{
				FilePath:  *file,
				OutputDir: *file + "-output", // do not forget this.
				Passwords: []string{pass},
				DirMode:   os.FileMode(0750),
				FileMode:  os.FileMode(0750),
			}

			// size is how many bytes were written.
			// files may be nil, but will contain any files written (even with an error).
			//size, files, archives, err := xtractr.ExtractFile(x)
			size, _, _, err := xtractr.ExtractFile(x)
			if err == nil && size > 0 {
				log.Infof("Password: '%v'. Success.", pass)
				doneChan <- true
			}
			//log.Debugf("try %v: %v", index, pass)
			progressBar.Increment()
		}(index, pass)
	}
	wg.Wait()
	log.Infof("duration %v", time.Since(startTime))
}
