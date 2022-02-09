package main

import (
	"fmt"
	"math/big"
	"math/rand"
	"runtime"
	"time"
)

type Factorizer struct {
	DebugWorkerIsSleepy int     //required: 0
	WaitPrevious        int     //required: 20
	WaitThreshold       float64 //required: 1.1
	DebugForceCpuCount  int     //required: 0
	TasksToGenerate     int     //required: 100
	MinInput            int     //required: 3
	MaxInput            int     //required: 1000
	DebugDontFactor     bool    //required: false
}

func NewFactorizer() *Factorizer {
	app := new(Factorizer)

	app.DebugWorkerIsSleepy = 0
	app.WaitPrevious = 20
	app.WaitThreshold = 1.1
	app.DebugForceCpuCount = 0
	app.TasksToGenerate = 100
	app.MinInput = 3
	app.MaxInput = 1000
	app.DebugDontFactor = false

	return app
}

func (context *Factorizer) AsyncWorker(n int64, stopSignal chan bool, output chan<- *big.Int) {
	if n < 0 {
		panic(fmt.Sprintf("n! cannot be computed for numbers<0, like %d!", n))
	}

	result := big.NewInt(1)
	//result.MulRange(1, int64(n))  //easy, but non-stoppable solution

	for i := int64(1); i <= n; i++ {
		select {
		case <-stopSignal:
			return
		default:
			result.Mul(result, big.NewInt(i))
		}
	}

	if context.DebugWorkerIsSleepy > 0 {
		sleep := rand.Intn(context.DebugWorkerIsSleepy)
		time.Sleep(time.Duration(sleep) * time.Millisecond)
	}

	if context.DebugDontFactor {
		output <- big.NewInt(n) // DEBUG-ONLY
		return
	}

	output <- result
}

func (context *Factorizer) asyncWorkerJobRunner(inputChan <-chan ChanItem, workerId int) {
	executionDurationHistory := make([]time.Duration, context.WaitPrevious)
	executionCounter := 0

	//fmt.Printf("   [%d]  Worker ready\n", workerId)
	for inputEl := range inputChan {
		task, resolver := inputEl.task, inputEl.resolver
		//fmt.Printf("   [%d.%d] Processing %d! - %v\n", workerId, executionCounter, task, context.DEBUG_WORKER_IS_SLEEPY)

		stopSignalChannel := make(chan bool, 2) //TODO not sure why its needed
		resultChannel := make(chan *big.Int)
		executionLimit := time.Minute // heuristic, should not take longer than 1 minute for the first ~20 runs.
		if executionCounter >= context.WaitPrevious {
			executionLimit = time.Duration(float64(Sum(executionDurationHistory)) / float64(context.WaitPrevious) * context.WaitThreshold)

		}
		startTime := time.Now()

		go context.AsyncWorker(task, stopSignalChannel, resultChannel)

		select {
		case <-time.After(executionLimit):
			stopSignalChannel <- true

			res := big.NewInt(0)
			//fmt.Printf("   [%d.%d] Timed out: %d! = %d (t=%.4f sec, <=%.4f)\n", workerId, executionCounter, task, res, float64(executionDurationHistory[executionCounter%context.WAIT_PREVIOUS])/float64(time.Second), float64(executionLimit)/float64(time.Second))

			go func() { resolver <- res }()

		case res := <-resultChannel:
			executionDurationHistory[executionCounter%context.WaitPrevious] = time.Since(startTime)

			//fmt.Printf("    [%d.%d] Succeeded : %d! = %d (t=%.4f sec, <=%.4f)\n", workerId, executionCounter, task, res, float64(executionDurationHistory[executionCounter%context.WAIT_PREVIOUS])/float64(time.Second), float64(executionLimit)/float64(time.Second))

			executionCounter++

			go func() { resolver <- res }()
		}
	}
	//fmt.Printf("   [%d]  Worker done\n", workerId)

}

func (context *Factorizer) workerManager(maxWorkerCount int, inputChan <-chan int64, outputChan chan chan *big.Int) {
	inputChanProxy := make(chan ChanItem)
	for i := 0; i < maxWorkerCount; i++ {
		go context.asyncWorkerJobRunner(inputChanProxy, i)
	}

	go context.asyncTransformInputToIoChan(inputChan, outputChan, inputChanProxy)()
}

func (context *Factorizer) asyncTransformInputToIoChan(inputChan <-chan int64, outputChan chan chan *big.Int, inputChanProxy chan ChanItem) func() {
	return func() {
		for input := range inputChan {
			resolver := make(chan *big.Int)
			outputChan <- resolver
			inputChanProxy <- ChanItem{input, resolver}
		}
	}
}

func (context *Factorizer) getWorkerCount() int {
	if context.DebugForceCpuCount > 0 {
		return context.DebugForceCpuCount
	}

	return runtime.NumCPU() + 1
}

func (context *Factorizer) Start() {
	workerCount := context.getWorkerCount()

	inputChan := make(chan int64, context.TasksToGenerate)
	inputCopyChan := make(chan int64, context.TasksToGenerate)
	outputChanChan := make(chan chan *big.Int, workerCount)

	//fmt.Println("Starting..")
	context.workerManager(workerCount, inputChan, outputChanChan)

	//fmt.Println("Generating..")
	for i := 0; i < context.TasksToGenerate; i++ {
		input := context.generateInput()

		inputChan <- input
		inputCopyChan <- input
	}
	close(inputChan)
	close(inputCopyChan)

	//fmt.Println("Printing..")
	for i := 0; i < context.TasksToGenerate; i++ {
		result := <-<-outputChanChan
		input := <-inputCopyChan

		context.processResult(i, input, result)
	}

	//fmt.Println("Done..")
}

func (context *Factorizer) generateInput() (input int64) {
	input = int64(rand.Intn(context.MaxInput-context.MinInput) + context.MinInput)
	return
}

func (context *Factorizer) processResult(i int, input int64, result *big.Int) {
	fmt.Printf("%d.: %d! = %d, isOdd=%v, what a surprise\n", i+1, input, result, IsOdd(result))
}
