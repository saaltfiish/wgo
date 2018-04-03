package utils

import (
	"math"
	"strconv"
)

func Round(f float64, n int) float64 {
	pow10_n := math.Pow10(n)
	return math.Trunc((f+0.5/pow10_n)*pow10_n) / pow10_n
}

func ParseFloat(f float64) float64 {
	fo, _ := strconv.ParseFloat(strconv.FormatFloat(f, 'f', 2, 64), 64)
	return fo
}

// 判断是不是数字(整数&浮点)
func Numeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// 判断是不是整数
func IsInt(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
