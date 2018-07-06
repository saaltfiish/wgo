//
// lock.go
// Copyright (C) 2018 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package storage

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"wgo/utils"
)

const (
	LOCK_TIMEOUT = 60 // second
)

// get lock
func (s *Storage) GetLock(key string) (string, error) {
	if s != nil {
		// checksum
		cs := utils.NewUUID()
		// timestamp
		ts := time.Now().Unix()
		// lock value
		val := fmt.Sprint(ts, ",", cs)
		var tried int
		for tried <= 3 {
			tried++
			if err := s.Put(key, val, 0, true); err != nil { // true代表仅当key不存在时能set成功
				Debug("[GetLock] lock exists, key: %s, val: %s", key, val)
				// lock exist
				if ob := s.Get(key); ob != nil {
					//Debug("[GetLock][key: %s][old_val: %s]", key, old)
					old := string(ob.([]byte))
					vs := strings.SplitN(old, ",", 2)
					if ots, _ := strconv.Atoi(vs[0]); int(ts)-ots > LOCK_TIMEOUT { //过期
						//过期了,抢
						if glb, err := s.GetSet(key, val); err == nil {
							gl := string(glb.([]byte))
							if gl == old {
								//抢到了,返回checksum
								return cs, nil
							} else {
								//没抢到,并且还覆盖了人家抢到的锁,(可能会产生问题)
								//cc.Set(key, gl, 600*time.Second)
								Debug("[GetLock] be robbed")
							}
						} else { //奇怪的情况
							Debug("[GetLock][key: %s][old_val: %s][ots: %d][strange_failed]", key, old, ots)
							return "", err
						}
					} else {
						Debug("[GetLock][key: %s][old_val: %s][ots: %d][not_expired_failed]", key, old, ots)
					}
				} else { //奇怪的情况
					return "", err
				}
			} else {
				Debug("[GetLock][key: %s][val: %s][not_exists_ok]", key, val)
				return cs, nil
			}
		}
	}
	return "", fmt.Errorf("can't get lock")
}

// release lock
func (s *Storage) ReleaseLock(key string) error {
	if s != nil {
		if cur := s.Get(key); cur != nil {
			//vs := strings.SplitN(cur, ",", 2)
			Debug("[ReleaseLock][lock_val: %s]", cur)
			return s.Delete(key)
		}
	}
	return fmt.Errorf("release wrong")
}
