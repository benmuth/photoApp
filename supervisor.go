package main

import (
	//"os/signal"
	//"strings"
	//"fmt"
	//"net/http"
	//"os"

	"log"
	"os"
	"os/exec"
)

func main() {
	//arg 1 is the name of the program, the rest are the program's args
	cmd := exec.Command("./"+os.Args[1], os.Args[2:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
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
	// cmd.Process.Signal(syscall.SIGINT)
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
