package qparts

import (
	"fmt"
	"strconv"
	"strings"
)

func IncreasePortInAddress(addr string, index int) string {
	addrParts := strings.Split(addr, ":")
	port, _ := strconv.Atoi(addrParts[len(addrParts)-1])
	addrParts[len(addrParts)-1] = fmt.Sprintf("%d", port+index)
	return strings.Join(addrParts, ":")
}

func IndexOf(element int64, data []int64) int {
	for k, v := range data {
		if element == v {
			return k
		}
	}
	return -1 //not found.
}

func IndexOfMin(element int64, data []int64) int {
	for k, v := range data {
		if element == v {
			return k
		}

		if element > v && len(data) == 1 {
			return k
		}

		if element > v && k == len(data)-1 {
			return k
		}

		if element > v && k < len(data)-1 && element < data[k+1] {
			return k
		}
	}
	return -1 //not found.
}

func RemoveFromSlice(s []int64, i int64) []int64 {
	index := IndexOf(i, s)
	if index == -1 {
		Log.Error("Received index -1 for val ", i)
		return s
	}
	s[index] = s[len(s)-1]
	return s[:len(s)-1]
}

func RemoveFromSliceByIndex(s []int64, index int64) []int64 {
	// TODO: Optimization?
	return append(s[:index], s[index+1:]...)
	// s[index] = s[len(s)-1]
	// return s[:len(s)-1]
}

func Max64(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}

func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func Min64(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}

func CeilForce(x, y int64) int64 {
	res := x / y
	f := float64(x) / float64(y)
	if f > float64(res) {
		return res + 1
	} else {
		return res
	}
}

func CeilForceInt(x, y int) int {
	res := x / y
	f := float64(x) / float64(y)
	if f > float64(res) {
		return res + 1
	} else {
		return res
	}
}

func CeilForceInt64(x, y int64) int64 {
	res := x / y
	f := float64(x) / float64(y)
	if f > float64(res) {
		return res + 1
	} else {
		return res
	}
}

func AppendIfMissing(slice []int64, i int64) []int64 {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

func Sum(array []int64) int64 {
	var result int64 = 0
	for _, v := range array {
		result += v
	}
	return result
}

func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
