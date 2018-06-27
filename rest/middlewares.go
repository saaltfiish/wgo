// Package rest provides ...
package rest

import (
	"encoding/json"
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
				rest.SetAction(ACTION_CREATE)
			case "PATCH":
				rest.SetAction(ACTION_UPDATE)
			case "DELETE":
				rest.SetAction(ACTION_DELETE)
			default:
				rest.SetAction(ACTION_READ)
			}
			if ca := rest.Options(CustomActionKey); ca != nil {
				if cas, ok := ca.(string); ok {
					rest.SetAction(cas)
				}
			}

			// get user id
			rest.SetUserID()

			// 处理起始时间
			rest.setTimeRangeFromStartEnd()

			// 参数表
			var ct int
			var p, pp string
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

			restError := next(c)

			// access info
			if ac := c.Access(); ac != nil {
				// endpoint
				if ep := rest.Options(EndpointKey); ep != nil && ep.(string) != "" {
					ac.Service.Endpoint = ep.(string)
				} else {
					Warn("not found endpoint")
				}

				// action
				ac.Service.Action = rest.action
				if rk := c.Param(RowkeyKey); rk != "" {
					ac.Service.RowKey = rk
				}

				// user info
				ac.Service.User.Id = rest.GetUserID()

				// new & old
				if la := rest.Options(LimitAccessKey); la == nil && rest.new != nil {
					// 如果设置了LimitAccess, 就不记录传入的body, 主要针对登录密码
					if _, ok := rest.new.(string); ok {
						ac.Service.New = rest.new.(string)
					} else if nb, err := json.Marshal(rest.new); err == nil {
						ac.Service.New = string(nb)
					}
				}
				if rest.older != nil {
					if ob, err := json.Marshal(rest.older); err == nil {
						ac.Service.Old = string(ob)
					}
				}

				// desc
				if d := rest.Options(DescKey); d != nil {
					if desc, ok := d.(string); ok {
						ac.Service.Desc = desc
					}
				}
			}

			return restError
		}

	}
}
