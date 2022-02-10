package main

import (
	"fmt"
	"math/big"
	"math/rand"
	"runtime"
	"time"
)

type Factorizer struct {
	DebugWorkerIsSleepy int     //default: 0; sleeps at max this number of ms after worker done
	WaitPrevious        int     //default: 20; average this number of previous jobs
	WaitThreshold       float64 //default: 1.1; wait for jobs for average plus this percent
	DebugForceCpuCount  int     //default: 0; override cpu/worker count. 0 for no override
	TasksToGenerate     int     //default: 100; number of tasks to generate during Start
	MinInput            int     //default: 3; min number to factorize during Start
	MaxInput            int     //default: 1000; max number to factorize during Start
	DebugDontFactor     bool    //default: false; if true, the factorization will result with n, instead of n!
	DebugTrace          bool    //default: false; prints extra debug/trace information
}

// NewFactorizer is the  default constructor for Factorizer.
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

// asyncWorker calculates `n!`, and writes it to `outputChan`. Can be killed with `stopSignal`.
// debug options:
//  -  context.DebugWorkerIsSleepy (artifical waiting time)
//  -  context.DebugDontFactor (after calculation/waiting, return with the original `n`, instead of `n!`
// panics on n<0, since factorial is mathematically defined only on positive numbers.
func (context *Factorizer) asyncWorker(n int64, stopSignal chan bool, outputChan chan<- *big.Int) {
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
		outputChan <- big.NewInt(n) // DEBUG-ONLY
		return
	}

	outputChan <- result
}

// asyncWorkerJobRunner takes an `inputOutputChan ChanItem`, that have both a "task", and a channel, waiting for the result.
// Processes anything available from inputOutputChan. Keeps the runtime of the last `context.WaitPrevious` runs, and after the context.WaitPrevious th run,
//   also kills the worker if its runtime exceeds the previous run's average (plus a context.WaitThreshold percentage.).
// Sends the worker's result (ie. n!) to the output channel, or `0` if the calculation timeouted.
func (context *Factorizer) asyncWorkerJobRunner(inputOutputChan <-chan ChanItem, workerId int) {
	executionDurationHistory := make([]time.Duration, context.WaitPrevious)
	executionCounter := 0

	if context.DebugTrace {
		fmt.Printf("   [%d]  Worker ready\n", workerId)
	}
	for inputEl := range inputOutputChan {
		task, resolver := inputEl.task, inputEl.resolver
		if context.DebugTrace {
			fmt.Printf("   [%d.%d] Processing %d! - %v\n", workerId, executionCounter, task, context.DebugWorkerIsSleepy)
		}

		stopSignalChannel := make(chan bool, 2) //TODO not sure why its needed
		resultChannel := make(chan *big.Int)
		executionLimit := time.Minute // heuristic, should not take longer than 1 minute for the first ~20 runs.
		if executionCounter >= context.WaitPrevious {
			executionLimit = time.Duration(float64(Sum(executionDurationHistory)) / float64(context.WaitPrevious) * context.WaitThreshold)

		}
		startTime := time.Now()

		go context.asyncWorker(task, stopSignalChannel, resultChannel)

		select {
		case <-time.After(executionLimit):
			stopSignalChannel <- true

			res := big.NewInt(0)
			if context.DebugTrace {
				fmt.Printf("   [%d.%d] Timed out: %d! = %d (t=%.4f sec, <=%.4f)\n", workerId, executionCounter, task, res, float64(executionDurationHistory[executionCounter%context.WaitPrevious])/float64(time.Second), float64(executionLimit)/float64(time.Second))
			}

			go func() { resolver <- res }()

		case res := <-resultChannel:
			executionDurationHistory[executionCounter%context.WaitPrevious] = time.Since(startTime)

			if context.DebugTrace {
				fmt.Printf("    [%d.%d] Succeeded : %d! = %d (t=%.4f sec, <=%.4f)\n", workerId, executionCounter, task, res, float64(executionDurationHistory[executionCounter%context.WaitPrevious])/float64(time.Second), float64(executionLimit)/float64(time.Second))
			}

			executionCounter++

			go func() { resolver <- res }()
		}
	}
	if context.DebugTrace {
		fmt.Printf("   [%d]  Worker done\n", workerId)
	}

}

// StartWorkers starts multiple workerJobs asynchronously, and passes a transformed input to them.
//   transformation helps keeping the output in an ordered manner, however the usage is now `<-<-output` instead of `<-output`.
// Note: Buffer on input/output channels are advised.
func (context *Factorizer) StartWorkers(maxWorkerCount int, inputChan <-chan int64, outputChan chan chan *big.Int) {
	inputOutputChan := make(chan ChanItem, maxWorkerCount)
	for i := 0; i < maxWorkerCount; i++ {
		go context.asyncWorkerJobRunner(inputOutputChan, i)
	}

	go context.asyncTransformInputToIoChan(inputChan, outputChan, inputOutputChan)()
}

// asyncTransformInputToIoChan transform an input channel to an inputOutput channel, that consist of a task, and a resolver channel.
// The resolver then available on the outputChannel.
//   Basically `(<-inputOutput).resolver <- X` is available through `X <-<- outputChan`, and
//   `inputChan <- X` is available at `X (<-inputOutput).task`.
func (context *Factorizer) asyncTransformInputToIoChan(inputChan <-chan int64, outputChan chan chan *big.Int, inputOutputChan chan ChanItem) func() {
	return func() {
		for input := range inputChan {
			resolver := make(chan *big.Int)
			outputChan <- resolver
			inputOutputChan <- ChanItem{input, resolver}
		}
	}
}

// getWorkerCount gets the worker count for factorizer. Calculated as `NumCPU+1`. Over-writable with debug option context.DebugForceCpuCount.
func (context *Factorizer) getWorkerCount() int {
	if context.DebugForceCpuCount > 0 {
		return context.DebugForceCpuCount
	}

	return runtime.NumCPU() + 1
}

// Start the factorization process. Default entry point, however It's possible to start with StartWorkers too.
// - calls StartWorkers, with the calculated worker count, a buffered input and output channel.
// - generates some inputs, and passes to input channel
// - reads the output channel, and prints the result.
func (context *Factorizer) Start() {
	workerCount := context.getWorkerCount()

	inputChan := make(chan int64, workerCount)
	inputCopyChan := make(chan int64, workerCount)
	outputChanChan := make(chan chan *big.Int, workerCount)

	context.StartWorkers(workerCount, inputChan, outputChanChan)
	done := make(chan bool, 2)
	go func() {
		for i := 0; i < context.TasksToGenerate; i++ {
			input := context.generateInput()

			inputChan <- input
			inputCopyChan <- input
		}
		close(inputChan)
		close(inputCopyChan)
		done <- true
	}()

	go func() {

		for i := 0; i < context.TasksToGenerate; i++ {
			result := <-<-outputChanChan
			input := <-inputCopyChan

			context.processResult(i, input, result)
		}
		done <- true
	}()

	<-done
	<-done

}

// generateInput generates a random number between MinInput and MaxInput (inclusive).
func (context *Factorizer) generateInput() (input int64) {
	input = int64(rand.Intn(context.MaxInput-context.MinInput) + context.MinInput)
	return
}

// processResult prints the result formatted, and determines the parity of the result.
func (context *Factorizer) processResult(i int, input int64, result *big.Int) {
	fmt.Printf("%d.: %d! = %d, isOdd=%v, what a surprise\n", i+1, input, result, IsOdd(result))
}
