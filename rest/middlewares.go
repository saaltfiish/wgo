// Package rest provides ...
package rest

import (
	"strconv"
	"strings"

	"wgo"
	"wgo/wcache"
	"wgo/whttp"
)

// REST
// 解析参数
func Init() wgo.MiddlewareFunc {
	cache = wcache.NewCache()
	//wcache.SetLogger(wgo)
	return func(next wgo.HandlerFunc) wgo.HandlerFunc {
		return func(c *wgo.Context) (err error) {
			rest := NewREST(c)
			defer rest.Release()

			// action
			switch m := c.Request().(whttp.Request).Method(); m {
			case "POST", "PUT":
				rest.SetAction("creating")
			case "PATCH":
				rest.SetAction("updating")
			default:
			}
			var ct int
			var p, pp string

			// user id
			uid := c.UserID()
			if _, err := strconv.Atoi(uid); err == nil {
				// 目前只支持数字类型的userid
				rest.SetEnv(USERID_KEY, c.UserID())
			}

			// 处理起始时间
			rest.setTimeRangeFromStartEnd()

			// 参数表
			params := c.QueryParams()
			for k, v := range params {
				switch k { //处理参数
				case PARAM_START, PARAM_END:
					continue
				case PARAM_DATE:
					rest.setTimeRangeFromDate(v)
				case PARAM_ORDERBY:
					rest.setOrderBy(v)
				case PARAM_FIELDS:
					//过滤字段
					if len(v) > 1 { //传了多个
						rest.SetEnv(FieldsKey, v)
					} else {
						if strings.Contains(v[0], ",") {
							rest.SetEnv(FieldsKey, strings.Split(v[0], ","))
						} else {
							rest.SetEnv(FieldsKey, v)
						}
					}
				case PARAM_PERPAGE:
					if len(v) > 0 {
						pp = v[0]
					}
				case PARAM_PAGE: //分页信息
					if len(v) > 0 {
						p = v[0]
					}
				default:
					//除了以上的特别字段,其他都是条件查询
					var cv interface{}

					if len(v) > 1 {
						cv = v
					} else {
						if strings.Contains(v[0], ",") {
							cv = strings.Split(v[0], ",")
						} else {
							cv = v[0]
						}
					}

					//根据参数名第一个字符来判断条件类型
					prefix := k[0] //param prefix
					if ct = getCTypeByPrefix(prefix); ct != CTYPE_IS {
						k = k[1:]
						//Debug("[key: %s][ctype: %d]", k, ct)
					}
					k = strings.TrimPrefix(k, "_")

					if strings.Contains(k, "|") { //包含"|",OR条件
						os := strings.Split(k, "|")
						for _, of := range os {
							if of != "" {
								//c.Info("[or condition][or field: %s]", of)
								rest.setCondition(NewCondition(CTYPE_OR, of, NewCondition(ct, k, cv))) // k代表同类的or条件
							}
						}
					} else {
						//如果参数中包含".",代表有关联查询
						if strings.Contains(k, ".") {
							js := strings.SplitN(k, ".", 2)
							if js[0] != "" && js[1] != "" {
								k = js[0]
								cv = NewCondition(ct, js[1], cv)
								//查询类型变为join
								c.Info("join: %s, %s; con: %v", k, cv.(*Condition).Field, cv)
								ct = CTYPE_JOIN
							}
						}
						rest.setCondition(NewCondition(ct, k, cv))
					}
				}
			}
			//记录分页信息
			rest.SetEnv(PaginationKey, NewPagination(p, pp))

			return next(c)
		}

	}
}
