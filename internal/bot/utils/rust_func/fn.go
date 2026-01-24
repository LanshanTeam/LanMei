package rust_func

/*
#cgo LDFLAGS: -L../lib/wcloud/target/release/ -lwcloud -lm
#include "../lib/wcloud/target/release/libwcloud.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func _Cadd() {
	result := C.add(5, 3)
	fmt.Println("Result from Rust:", result)
}

func Wcloud(words map[string]int64) string {
	cWords := make([]C.Word, len(words))
	i := 0
	for k, v := range words {
		cWords[i] = C.Word{
			word: C.CString(k),
			freq: C.size_t(v),
		}
		i++
	}
	defer func() {
		for _, w := range cWords {
			C.free(unsafe.Pointer(w.word))
		}
	}()
	// 调用Rust函数
	result := C.wcloud(&cWords[0], C.int(len(cWords)))
	defer C.free_string(result)
	// 将结果转换为Go字符串
	return C.GoString(result)
}
