package bulky

import (
	"context"
	"sync"
	"time"
)

//ProcessEvent contract
type ProcessEvent interface {
	OnProcess([]Data) error
	OnProcessError(string, []Data)
	OnProcessTimeout([]Data)
	OnScheduleFailed([]Data)
}

//Option as configuration
type Option struct {
	MaxInFlight          int
	MaxScheduledProcess  int
	NumberOfDataAtOnce   int
	ProcessTimeoutSecond int
}

//Data holds passed data
type Data struct {
	Body interface{}
}

//ErrEvent error event
type ErrEvent struct {
	Err  error
	Args []interface{}
}

//BulkDataProcessor structure
type BulkDataProcessor struct {
	nbulk    int
	nchan    int
	nflight  int
	nproc    *nproc
	run      bool
	buff     *buff
	dschan   chan []Data
	event    ProcessEvent
	ptimeout int
}

type nproc struct {
	count int
	sync.RWMutex
}

type buff struct {
	data []Data
	sync.RWMutex
}

//NewBulkDataProcessor creates new bulk data processor
func NewBulkDataProcessor(event ProcessEvent, option Option) *BulkDataProcessor {
	proc := &BulkDataProcessor{
		nbulk:    option.NumberOfDataAtOnce,
		nchan:    option.MaxScheduledProcess,
		nflight:  option.MaxInFlight,
		run:      true,
		event:    event,
		dschan:   make(chan []Data, 1),
		buff:     new(buff),
		nproc:    new(nproc),
		ptimeout: option.ProcessTimeoutSecond,
	}
	if proc.nchan > 0 {
		proc.dschan = make(chan []Data, proc.nchan)
	}
	return proc
}

//Process processes the scheduled data
func (b *BulkDataProcessor) Process(stop chan<- bool) {
	for {
		if b.nproc.count >= b.nflight {
			continue
		}
		select {
		case dds := <-b.dschan:
			b.nproc.count++
			go b.do(dds)
		default: // no schedule
		}
		//clear remain data
		if !b.run {
			b.cleanup()
			break
		}
	}
	//wait all done
	for b.nproc.count != 0 {
	}
	stop <- true
	close(stop)
}

func (b *BulkDataProcessor) cleanup() {
	if b.buff != nil && b.buff.data != nil {
		b.nproc.count++
		go b.do(b.buff.data)
	}
	var rproc int
	rem := len(b.dschan)
	for {
		if rproc == rem {
			break
		}
		select {
		case dds := <-b.dschan:
			b.nproc.count++
			rproc++
			go b.do(dds)
		default:
			break
		}
	}
	b.buff = nil
}

//Schedule schedules the data
func (b *BulkDataProcessor) Schedule(data Data) {
	if b.buff.data == nil {
		b.buff.data = make([]Data, 0)
	}
	b.buff.Lock()
	defer b.buff.Unlock()
	b.buff.data = append(b.buff.data, data)
	if len(b.buff.data) == b.nbulk {
		select {
		case b.dschan <- b.buff.data:
		default:
			b.event.OnScheduleFailed(b.buff.data)
		}
		b.buff.data = nil
	}
}

//Stop set runnier to false
func (b *BulkDataProcessor) Stop() {
	b.run = false
}

//ConsumeBuffer forces process on the event using existing buffer data
func (b *BulkDataProcessor) ConsumeBuffer() {
	b.buff.Lock()
	defer b.buff.Unlock()
	if len(b.buff.data) == 0 {
		return
	}
	b.do(b.buff.data)
	b.buff.data = nil
}

//do runs the process
func (b *BulkDataProcessor) do(ss []Data) {
	defer func() {
		b.nproc.Lock()
		defer b.nproc.Unlock()
		b.nproc.count--
	}()

	tctx, cancel := context.WithTimeout(context.Background(),
		time.Second*time.Duration(b.ptimeout))
	defer cancel()

	echan := make(chan error, 1)
	go func(colls []Data) {
		echan <- b.event.OnProcess(colls)
	}(ss)
	select {
	case <-tctx.Done():
		b.event.OnProcessTimeout(ss)
	case err := <-echan:
		if err != nil {
			b.event.OnProcessError(err.Error(), ss)
		}
	}
}
