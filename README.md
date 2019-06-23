# Bulky - Bulk Data Processor

This library schedule the data stream, and process it at once after collecting n data. This can be used for pub-sub operation to support MaxInFlight (number of worker).

## Usage

To use this library, we need to create an `Event` and `Processor` object.

### Event

```go
type event struct{}

func (e *event) OnProcess(colls []bulky.Data) error {
    fmt.Println("on processing: ", colls)
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
```

There are two event type:

1. **OnProcess**: is a callback when the bulk data is collected

2. **OnProcessFailed**: will be called when the process above return error

3. **OnProcessTimeout**: when the process exceed the deadline, this callback will be called

4. **OnScheduleFailed**: in case of the buffer of process is full, the library will returns the last n bulk data to be handled (i.e: re-published or re-queue)

### Processor

Initializing the processor by adding the event object and its option

```go
processor := bulky.NewBulkDataProcessor(new(event), bulky.Option{
    MaxInFlight:         10,
    MaxScheduledProcess: 10,
    NumberOfDataAtOnce:  20,
    ProcessTimeoutSecond: 1,
})
```

Options:

1. `MaxInFlight`: Number of concurrent process at once taken from scheduler channel (worker)

2. `MaxScheduledProcess`: Maximum capacity of scheduled process (pending process)

3. `NumberOrDataAtOnce`: Number of bulk data to be processed. The data will be process after `n` data is collected

For example, we can create a simple scheduler to schedule a stream of data:

```go
var seeds []int //sample data from 1 - 90

go func(is []int) {
    for _, i := range is {
        processor.Schedule(bulky.Data{Body: i}) //schedule to process
    }
    processor.Stop() //stop after done
}(seeds)

stopChan := make(chan bool, 1)
go processor.Process(stopChan)

<-stopChan //wait till finish
fmt.Println("finish")
```

It will result like this:

```txt
on processing:  [{0} {1} {2} {3} {4} {5} {6} {7} {8} {9} {10} {11} {12} {13} {14} {15} {16} {17} {18} {19}]
on processing:  [{80} {81} {82} {83} {84} {85} {86} {87} {88} {89} {90}] //remain data, handled last after remain schedule done
on processing:  [{40} {41} {42} {43} {44} {45} {46} {47} {48} {49} {50} {51} {52} {53} {54} {55} {56} {57} {58} {59}]
on processing:  [{20} {21} {22} {23} {24} {25} {26} {27} {28} {29} {30} {31} {32} {33} {34} {35} {36} {37} {38} {39}]
on processing:  [{60} {61} {62} {63} {64} {65} {66} {67} {68} {69} {70} {71} {72} {73} {74} {75} {76} {77} {78} {79}]
```
