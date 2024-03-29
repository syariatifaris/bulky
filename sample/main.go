package main

import (
	"fmt"
	"time"

	"github.com/syariatifaris/bulky"
)

type event struct{}

func (e *event) OnProcess(colls []bulky.Data) error {
	fmt.Println("on processing: ", colls)
	time.Sleep(time.Second / 2)
	return nil
}

func (e *event) OnProcessTimeout(colls []bulky.Data) {
	fmt.Println("process timeout:", colls)
}

func (e *event) OnProcessError(cause string, colls []bulky.Data) {
	fmt.Println("process error:", colls, "cause:", cause)
}

func (e *event) OnScheduleFailed(colls []bulky.Data) {
	fmt.Println("on schedule failed: ", colls)
}

func main() {
	seeds := makeRange(0, 99)
	processor := bulky.NewBulkDataProcessor(new(event), bulky.Option{
		MaxInFlight:          10,
		MaxScheduledProcess:  10,
		NumberOfDataAtOnce:   10,
		ProcessTimeoutSecond: 1,
	})

	start := time.Now()
	//simple schedule
	go func(is []int) {
		for _, i := range is {
			processor.Schedule(bulky.Data{Body: i})
		}
		processor.Stop()
	}(seeds)

	stopChan := make(chan bool, 1)
	go processor.Process(stopChan)
	<-stopChan //wait till finish
	fmt.Println("finish, process time: ", time.Since(start).Nanoseconds())
}

func makeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}
