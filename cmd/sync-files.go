package cmd

import (
	"fmt"
	"github.com/briandowns/spinner"
	"github.com/zloylos/grsync"
	"time"
)

func syncFiles(destination string, filesPath string, remoteUser string) {
	fmt.Printf("Syncing files from server %v and path %v\n", destination, filesPath)

	task := grsync.NewTask(
		fmt.Sprintf("%v@%v:%v", remoteUser, destination, filesPath),
		"./",
		grsync.RsyncOptions{},
	)

	go func() {
		s := spinner.New(spinner.CharSets[43], 150*time.Millisecond)
		s.Start()
	}()

	if err := task.Run(); err != nil {
		panic(err)
	}

	fmt.Println("\nEverything is synced!")
}
