package bulky

//ProcessEvent contract
type ProcessEvent interface {
	OnProcess([]Data)
	OnScheduleFailed([]Data)
}

//Option as configuration
type Option struct {
	MaxInFlight         int
	MaxScheduledProcess int
	NumberOfDataAtOnce  int
}

//Data holds passed data
type Data struct {
	Body interface{}
}

//BulkDataProcessor structure
type BulkDataProcessor struct {
	nbulk   int
	nchan   int
	nflight int
	nproc   int
	run     bool
	buff    []Data
	dschan  chan []Data
	event   ProcessEvent
}

//NewBulkDataProcessor creates new bulk data processor
func NewBulkDataProcessor(event ProcessEvent, option Option) *BulkDataProcessor {
	proc := &BulkDataProcessor{
		nbulk:   option.NumberOfDataAtOnce,
		nchan:   option.MaxScheduledProcess,
		nflight: option.MaxInFlight,
		run:     true,
		event:   event,
		dschan:  make(chan []Data, 1),
	}
	if proc.nchan > 0 {
		proc.dschan = make(chan []Data, proc.nchan)
	}
	return proc
}

//Process processes the scheduled data
func (b *BulkDataProcessor) Process(stop chan<- bool) {
	for {
		if b.nproc >= b.nflight {
			continue
		}
		select {
		case dds := <-b.dschan:
			b.nproc++
			go b.do(&b.nproc, dds)
		default: // no schedule
		}
		//clear remain data
		if !b.run {
			b.cleanup()
			break
		}
	}
	//wait all done
	for b.nproc != 0 {
	}
	stop <- true
	close(stop)
}

func (b *BulkDataProcessor) cleanup() {
	if b.buff != nil {
		b.nproc++
		go b.do(&b.nproc, b.buff)
	}
	var rproc int
	rem := len(b.dschan)
	for {
		if rproc == rem {
			break
		}
		select {
		case dds := <-b.dschan:
			b.nproc++
			rproc++
			go b.do(&b.nproc, dds)
		default:
			break
		}
	}
	b.buff = nil
}

//Schedule schedules the data
func (b *BulkDataProcessor) Schedule(data Data) error {
	if b.buff == nil {
		b.buff = make([]Data, 0)
	}
	b.buff = append(b.buff, data)
	if len(b.buff) == b.nbulk {
		select {
		case b.dschan <- b.buff:
		default:
			b.event.OnScheduleFailed(b.buff)
		}
		b.buff = nil
	}
	return nil
}

//Stop set runnier to false
func (b *BulkDataProcessor) Stop() {
	b.run = false
}

//do runs the process
func (b *BulkDataProcessor) do(nproc *int, ss []Data) {
	defer func() {
		*nproc = *nproc - 1
	}()
	b.event.OnProcess(ss)
}