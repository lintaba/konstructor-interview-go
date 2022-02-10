package main

// #include <string.h>
// // complex oddity check
// int isOdd(char* num)
// {
//	    return (num[strlen(num)-1] - '0') % 2;
// }
// // simple oddity check. (the one and only odd factorial number..)
// int isOddSimple(char* num){
//      return strcmp(num, "1")==0;
// }
import "C"
import "math/big"

// IsOdd determines if a number is odd (ie 1, 3). Returns `false` for even numbers (ie 0, 2)
//    Uses a C module as internal implementation.
func IsOdd(num *big.Int) bool {

	return C.isOdd(C.CString(num.String())) > 0
}
