package utils

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"time"
)

type reducetype func(interface{}) interface{}
type filtertype func(interface{}) bool

// InSlice checks given string in string slice or not.
func InSliceIgnorecase(v string, sl []string) bool {
	for _, vv := range sl {
		if strings.ToLower(vv) == strings.ToLower(v) {
			return true
		}
	}
	return false
}

// InSlice checks given string in string slice or not.
func InSlice(v string, sl []string) bool {
	for _, vv := range sl {
		if vv == v {
			return true
		}
	}
	return false
}

// InSliceInt checks given int in int slice or not.
func InSliceInt(v int, sl []int) bool {
	for _, vv := range sl {
		if vv == v {
			return true
		}
	}
	return false
}

// InSliceInt64 checks given int64 in int64 slice or not.
func InSliceInt64(v int64, sl []int64) bool {
	for _, vv := range sl {
		if vv == v {
			return true
		}
	}
	return false
}

// Purge Slice
func PurgeSlice(v string, sl []string) []string {
	nsl := []string{}
	for _, vv := range sl {
		if vv != v {
			nsl = append(nsl, vv)
		}
	}
	return nsl
}

// Purge Slice Int
func PurgeSliceInt(v int, sl []int) []int {
	nsl := []int{}
	for _, vv := range sl {
		if vv != v {
			nsl = append(nsl, vv)
		}
	}
	return nsl
}

// Purge Slice Int64
func PurgeSliceInt64(v int64, sl []int64) []int64 {
	nsl := []int64{}
	for _, vv := range sl {
		if vv != v {
			nsl = append(nsl, vv)
		}
	}
	return nsl
}

// InSliceIface checks given interface in interface slice.
func InSliceIface(v interface{}, sl []interface{}) bool {
	for _, vv := range sl {
		if vv == v {
			return true
		}
	}
	return false
}

// SliceRandList generate an int slice from min to max.
func SliceRandList(min, max int) []int {
	if max < min {
		min, max = max, min
	}
	length := max - min + 1
	t0 := time.Now()
	rand.Seed(int64(t0.Nanosecond()))
	list := rand.Perm(length)
	for index, _ := range list {
		list[index] += min
	}
	return list
}

// SliceMerge merges interface slices to one slice.
func SliceMerge(slice1, slice2 []interface{}) (c []interface{}) {
	c = append(slice1, slice2...)
	return
}

// SliceReduce generates a new slice after parsing every value by reduce function
func SliceReduce(slice []interface{}, a reducetype) (dslice []interface{}) {
	for _, v := range slice {
		dslice = append(dslice, a(v))
	}
	return
}

// SliceRand returns random one from slice.
func SliceRand(a []interface{}) (b interface{}) {
	randnum := rand.Intn(len(a))
	b = a[randnum]
	return
}

// SliceSum sums all values in int64 slice.
func SliceSum(intslice []int64) (sum int64) {
	for _, v := range intslice {
		sum += v
	}
	return
}

// SliceFilter generates a new slice after filter function.
func SliceFilter(slice []interface{}, a filtertype) (ftslice []interface{}) {
	for _, v := range slice {
		if a(v) {
			ftslice = append(ftslice, v)
		}
	}
	return
}

// SliceDiff returns diff slice of slice1 - slice2.
func SliceDiff(slice1, slice2 []interface{}) (diffslice []interface{}) {
	for _, v := range slice1 {
		if !InSliceIface(v, slice2) {
			diffslice = append(diffslice, v)
		}
	}
	return
}

// SliceIntersect returns diff slice of slice2 - slice1.
func SliceIntersect(slice1, slice2 []interface{}) (diffslice []interface{}) {
	for _, v := range slice1 {
		if !InSliceIface(v, slice2) {
			diffslice = append(diffslice, v)
		}
	}
	return
}

// SliceChuck separates one slice to some sized slice.
func SliceChunk(slice []interface{}, size int) (chunkslice [][]interface{}) {
	if size >= len(slice) {
		chunkslice = append(chunkslice, slice)
		return
	}
	end := size
	for i := 0; i <= (len(slice) - size); i += size {
		chunkslice = append(chunkslice, slice[i:end])
		end += size
	}
	return
}

// SliceRange generates a new slice from begin to end with step duration of int64 number.
func SliceRange(start, end, step int64) (intslice []int64) {
	for i := start; i <= end; i += step {
		intslice = append(intslice, i)
	}
	return
}

// SlicePad prepends size number of val into slice.
func SlicePad(slice []interface{}, size int, val interface{}) []interface{} {
	if size <= len(slice) {
		return slice
	}
	for i := 0; i < (size - len(slice)); i++ {
		slice = append(slice, val)
	}
	return slice
}

// SliceUnique cleans repeated values in slice.
func SliceUnique(slice []interface{}) (uniqueslice []interface{}) {
	for _, v := range slice {
		if !InSliceIface(v, uniqueslice) {
			uniqueslice = append(uniqueslice, v)
		}
	}
	return
}

// SliceShuffle shuffles a slice.
func SliceShuffle(slice []interface{}) []interface{} {
	for i := 0; i < len(slice); i++ {
		a := rand.Intn(len(slice))
		b := rand.Intn(len(slice))
		slice[a], slice[b] = slice[b], slice[a]
	}
	return slice
}

// toSliceType returns the element type of the given object, if the object is a
// "*[]*Element" or "*[]Element". If not, returns nil.
// err is returned if the user was trying to pass a pointer-to-slice but failed.
func toSliceType(i interface{}) (reflect.Type, error) {
	t := reflect.TypeOf(i)
	if t.Kind() != reflect.Ptr {
		// If it's a slice, return a more helpful error message
		if t.Kind() == reflect.Slice {
			return nil, fmt.Errorf("utils: this is non-pointer slice: %v", t)
		}
		return nil, nil
	}
	if t = t.Elem(); t.Kind() != reflect.Slice {
		return nil, nil
	}
	return t.Elem(), nil
}
