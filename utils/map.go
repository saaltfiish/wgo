//
// map.go
// Copyright (C) 2019 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package utils

func MapKeys(m map[string]string) (keys []string) {
	if len(m) > 0 {
		for k, _:=range m{
			keys = append(keys, k)
		}
	}
	return
}
