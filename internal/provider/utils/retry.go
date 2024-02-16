package utils

import {
	"time"
}

func retry(toRetry func() error) error {
	n := 0
	var err error
	for n < 10 {
		err = toRetry()
		if err == nil {
			break
		}
		n++
		log.Printf("[DEBUG] Issue on request retry: %d, -> try again", n)
		sleepDuration, _ := time.ParseDuration(fmt.Sprintf("%ds", 2*n))
		time.Sleep(sleepDuration)
	}
	return err
}