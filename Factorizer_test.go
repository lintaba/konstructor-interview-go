package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"math/big"
	"testing"
)

func TestWorker(t *testing.T) {
	var expTests = []struct {
		n        int64  // input
		expected string // expected result
	}{
		{0, "1"},
		{1, "1"},
		{3, "6"},
		{30, "265252859812191058636308480000000"},
		{42, "1405006117752879898543142606244511569936384000000000"},
	}

	for _, tt := range expTests {
		i := new(big.Int)
		stopSign := make(chan bool)
		out := make(chan *big.Int)
		i.SetString(tt.expected, 10)
		fc := NewFactorizer()

		go fc.asyncWorker(tt.n, stopSign, out)

		output := <-out

		assert.Equal(t, i, output, "bad result for %d!", tt.n)
	}
}
func TestWorkerInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	fc := NewFactorizer()
	fc.asyncWorker(-1, make(chan bool), make(chan *big.Int))
}

func TestSlowWorker(t *testing.T) {
	fc := NewFactorizer()
	fc.WaitPrevious = 20 //to be sure..

	inputChan := make(chan ChanItem)
	go fc.asyncWorkerJobRunner(inputChan, 42)

	//expect 20 items normally.
	fc.DebugWorkerIsSleepy = 10
	for i := 0; i < fc.WaitPrevious; i++ {
		resolver := make(chan *big.Int)
		inputChan <- ChanItem{5, resolver}
		val := <-resolver
		assert.Equal(t, big.NewInt(120), val, "bad result at %v!", i)
	}
	{
		fc.DebugWorkerIsSleepy = 0
		//next should be still fast.
		resolver := make(chan *big.Int)
		inputChan <- ChanItem{3, resolver}
		val := <-resolver
		assert.Equal(t, big.NewInt(6), val, "bad result!")
	}
	{
		fc.DebugWorkerIsSleepy = 100
		resolver := make(chan *big.Int)
		inputChan <- ChanItem{4, resolver}
		val := <-resolver
		assert.Equal(t, big.NewInt(0), val, "timeout is not 0, but %v!", val)
	}
}

type MockFactor struct {
	// add a Mock object instance
	mock.Mock
	t *testing.T

	// other fields go here as normal
	Factorizer
}

//TODO, figure out how to do mocks, so TestOrderWorks wouldnt be a code duplication, and also would be tested.
//func (o *MockFactor) processResult(i int, input int64, result *big.Int) {
//
//	assert.Equal(o.t, int64(i), input)
//	assert.Equal(o.t, big.NewInt(int64(i)), result)
//}
//
//func TestOrderWorks2(t *testing.T) {
//
//	context := new(MockFactor)
//	context.DEBUG_FORCE_CPU_COUNT = 100
//	context.DEBUG_WORKER_IS_SLEEPY = 50
//	context.TASKS_TO_GENERATE = 1000
//	context.DEBUG_DONT_FACTOR = true
//
//	context.Start()
//
//}

func TestOrderWorks(t *testing.T) {

	context := NewFactorizer()
	context.DebugForceCpuCount = 100
	context.DebugWorkerIsSleepy = 50
	context.TasksToGenerate = 1000
	context.DebugDontFactor = true

	workerCount := context.getWorkerCount()

	inputChan := make(chan int64, context.TasksToGenerate)
	inputCopyChan := make(chan int64, context.TasksToGenerate)
	outputChanChan := make(chan chan *big.Int, workerCount)

	context.StartWorkers(workerCount, inputChan, outputChanChan)

	for i := 0; i < context.TasksToGenerate; i++ {
		var input = int64(i)

		inputChan <- input
		inputCopyChan <- input
	}

	for i := 0; i < context.TasksToGenerate; i++ {
		result := <-<-outputChanChan
		input := <-inputCopyChan

		assert.Equal(t, int64(i), input)
		assert.Equal(t, big.NewInt(int64(i)), result)
	}
}
