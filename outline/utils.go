package outline

import "fmt"

type ByteNum uint64

func (n ByteNum) String() (str string) {
	if n == 0 {
		str = "0 B"
		return
	}
	if n < 1024 {
		str = fmt.Sprintf("%v B", uint64(n))
		return
	}
	for _, format := range []string{"%.2f KB", "%.2f MB", "%.2f GB", "%.2f TB"} {
		if n > 0 && (n >> 20) == 0 {
			str = fmt.Sprintf(format, float64(n)/1024.0)
			break
		}
		n >>= 10
	}

	return
}
