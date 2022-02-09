package main

import (
	"math/big"
	"time"
)

func Sum(arr []time.Duration) (res time.Duration) {
	for _, el := range arr {
		res += el
	}
	return
}

type ChanItem struct {
	task     int64         //number to factorize
	resolver chan *big.Int //channel to push the factor when it's done
}
