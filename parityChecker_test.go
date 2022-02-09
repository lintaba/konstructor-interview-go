package main

import (
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestIsOdd(t *testing.T) {

	assert.Equal(t, true, IsOdd(big.NewInt(1)))
	assert.Equal(t, false, IsOdd(big.NewInt(2)))
	assert.Equal(t, false, IsOdd(big.NewInt(2000000)))
	assert.Equal(t, true, IsOdd(big.NewInt(2000001)))

}
