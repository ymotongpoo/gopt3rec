package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func BatchRec(path string) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		log.Fatalf("Path doesn't exist: %v", path)
	}

	fileCh := make(chan string, len(TVChannelMap))
	go func() {
		for _, v := range TVChannelMap {
			tsfile := filepath.Join(path, v+".ts")
			cmdStr := []string{"recpt1", "--b25", "--strip", v, "180", tsfile}
			cmd := exec.Command(cmdStr[0], cmdStr[1:])
			err = cmd.Run()
			if err != nil {
				log.Println("failed to record: %v", v)
				continue
			}
			fileCh <- tsfile
		}
		fileCh.Close()
	}()

	for tsfile := range fileCh {
		cmdStr := []string("epgdump", "json", tsfile, tsfile+".json")
		cmd := exec.Command(cmdStr[0], cmdStr[1:])
		err = cmd.Run()
	}
}
