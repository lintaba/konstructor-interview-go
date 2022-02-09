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

func IsOdd(num *big.Int) bool {

	return C.isOdd(C.CString(num.String())) > 0
}
