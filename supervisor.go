package main

import (
	//"os/signal"
	//"strings"
	//"fmt"
	//"net/http"
	//"os"

	"flag"
	"log"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	dirPath := flag.String("dirPath", "/Users/moose1/Documents/photoApp/", "designate directory path of executable")
	dbPath := flag.String("db", "/Users/moose1/Documents/photoApp/photoAppDB", "designate database path to use")
	flag.Parse()
	cmd := exec.Command(*dirPath+"photoApp", "-db "+*dbPath)
	cmd.Stdout = os.Stdout
	if err := cmd.Start(); err != nil {
		log.Printf("failed to start app: %s", err)
		return
	}

	/*
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			log.Printf("received signal: %v", sig)
			err := cmd.Process.Signal(sig)
			if err != nil {
				log.Printf("failed to execute signal %v: %s", sig, err)
				return
			}
		}()
	*/

	done := make(chan bool, 1)
	go func() {
		pState, err := cmd.Process.Wait()
		if err != nil {
			log.Printf("error on process exit")
			return
		}
		log.Printf("exited?: %v", pState.Exited())
		done <- true
	}()
	cmd.Process.Signal(syscall.SIGINT)
	<-done
	/*
		if pState.Exited() == true {
			client := http.Client{}
			message := fmt.Sprintf("exited+with+code+%v,+successfully?:+%v+;+%s", pState.ExitCode(), pState.Success(), pState.String())
			resp, err := client.Post("https://api.pushover.net/1/messages.json", "application/x-www-form-urlencoded", strings.NewReader("token=am38djsk2zp8q2d5eeqveiecddvoiu&user=uyfz5is338ugpgfxa75onm3heaq1kd&message="+message))
			if err != nil {
				log.Printf("failed to post message to pushover: %s", err)
				return
			}
			log.Printf("Pushover response: %v", resp.Body)
		}
	*/
}
