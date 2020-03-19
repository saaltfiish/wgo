package rest

import (
	"io/ioutil"
	"reflect"
)

type Action interface { // 把action定义为一个interface, 这样业务代码可以overwrite
	PreGet() (interface{}, error)  //获取前
	WillGet() (interface{}, error) //获取前hook
	OnGet() (interface{}, error)   //获取中
	DidGet() (interface{}, error)  //获取到hook
	PostGet() (interface{}, error) //获取后

	PreSearch() (interface{}, error)  // 搜索前
	WillSearch() (interface{}, error) // 搜索前hook
	OnSearch() (interface{}, error)   // 搜索
	DidSearch() (interface{}, error)  // 搜索完hook
	PostSearch() (interface{}, error) // 搜索后

	PreCreate() (interface{}, error)  // 创建前
	WillCreate() (interface{}, error) // 创建hook
	OnCreate() (interface{}, error)   // 创建中
	DidCreate() (interface{}, error)  // 创建hook
	PostCreate() (interface{}, error) // 创建后

	PreUpdate() (interface{}, error)  // 更新前
	WillUpdate() (interface{}, error) // hook
	OnUpdate() (interface{}, error)   // 更新中
	DidUpdate() (interface{}, error)  // hook
	PostUpdate() (interface{}, error) // 更新后

	PreDelete() (interface{}, error)  // 删除前
	OnDelete() (interface{}, error)   // 删除中
	PostDelete() (interface{}, error) // 删除后

	PreCheck() (interface{}, error)  // 搜索前
	OnCheck() (interface{}, error)   // 搜索中
	PostCheck() (interface{}, error) // 搜索后

	Trigger() (interface{}, error) //触发器
	Defer()
}

/* {{{ func (r *REST) Trigger(i interface{}) (interface{}, error)
 *
 */
func (r *REST) Trigger() (interface{}, error) {
	return r.Model(), nil
}

/* }}} */
/* {{{ func (r *REST) Defer()
 *
 */
func (r *REST) Defer() {
}

/* }}} */

/* {{{ func (r *REST) PreGet() (interface{}, error)
 *
 */
func (r *REST) PreGet() (interface{}, error) {
	m := r.Model()
	c := r.Context()
	// pk,放入条件
	if id := c.Param(RowkeyKey); id != "" {
		pk, _, _ := m.PKey()
		//c.Debug("[PreGet][pk: %s, id: %s]", pk, id)
		m.SetConditions(NewCondition(CTYPE_IS, pk, id))
	}
	// 从rcontext里获取条件
	if tr := r.GetEnv(TimeRangeKey); tr != nil { //时间段参数
		m.SetConditions(NewCondition(CTYPE_IS, TAG_TIMERANGE, tr.(*TimeRange)))
	}
	m.SetConditions(r.GetParamConds()...) // 获取参数条件
	// fields
	if fs := r.GetEnv(FieldsKey); fs != nil { //从context里面获取参数条件
		m.SetFields(fs.([]string)...)
	}
	return m, nil
}

/* }}} */
/* {{{ func (r *REST) WillGet() (interface{}, error)
 * React灵感,啥也不做,只等覆盖
 */
func (r *REST) WillGet() (interface{}, error) {
	return r.Model(), nil
}

/* }}} */
/* {{{ func (r *REST) OnGet() (interface{}, error)
 *
 */
func (r *REST) OnGet() (interface{}, error) {
	m := r.Model()
	if row, err := m.Row(); err == nil {
		// Info("[OnGet]row: %+v", row)
		// return r.setModel(row.(Model)), err
		// reflect方法给指针赋值，相当于 *x=y, // https://stackoverflow.com/questions/40060131/reflect-assign-a-pointer-struct-value
		reflect.ValueOf(m).Elem().Set(reflect.ValueOf(row).Elem())
		// 这里还需要setModel, 因为刚才的赋值导致m丢失了*REST
		return r.setModel(m), nil
	} else {
		return nil, err
	}
}

/* }}} */
/* {{{ func (r *REST) DidGet() (interface{}, error)
 * React灵感,啥也不做,只等覆盖
 */
func (r *REST) DidGet() (interface{}, error) {
	return r.Model(), nil
}

/* }}} */
/* {{{ func (r *REST) PostGet() (interface{}, error)
 *
 */
func (r *REST) PostGet() (interface{}, error) {
	return r.Model().Protect()
}

/* }}} */

/* {{{ func (r *REST) PreSearch() (interface{}, error)
 *
 */
func (r *REST) PreSearch() (interface{}, error) {
	m := r.Model()
	if m == nil {
		Info("[PreSearch]%s's model is nil", r.Name())
	}
	// 从r里获取条件
	if p := r.GetEnv(PaginationKey); p != nil { //排序
		m.SetPagination(p.(*Pagination))
	}
	if ob := r.GetEnv(OrderByKey); ob != nil { //排序
		m.SetConditions(NewCondition(CTYPE_ORDER, TAG_ORDERBY, ob.(*OrderBy)))
	}
	if tr := r.GetEnv(TimeRangeKey); tr != nil { //时间段参数
		m.SetConditions(NewCondition(CTYPE_RANGE, TAG_TIMERANGE, tr.(*TimeRange)))
	}
	m.SetConditions(r.GetParamConds()...) // 获取参数条件
	// fields
	if fs := r.GetEnv(FieldsKey); fs != nil { //从context里面获取参数条件
		m.SetFields(fs.([]string)...)
	}
	return m, nil
}

/* }}} */
/* {{{ func (r *REST) WillSearch() (interface{}, error)
 * React灵感,啥也不做,只等覆盖
 */
func (r *REST) WillSearch() (interface{}, error) {
	return r.Model(), nil
}

/* }}} */
/* {{{ func (r *REST) OnSearch() (interface{}, error)
 *
 */
func (r *REST) OnSearch() (interface{}, error) {
	list, err := r.Model().List()
	if err != nil {
		return nil, err
	}
	return r.SetResult(list)
}

/* }}} */
/* {{{ func (r *REST) DidSearch() (interface{}, error)
 * React灵感,啥也不做,只等覆盖
 */
func (r *REST) DidSearch() (interface{}, error) {
	return r.Result(), nil
}

/* }}} */
/* {{{ func (r *REST) PostSearch() (interface{}, error)
 *
 */
func (r *REST) PostSearch() (interface{}, error) {
	return r.Result(), nil
}

/* }}} */

/* {{{ func (r *REST) PreCreate() (interface{}, error)
 *
 */
func (r *REST) PreCreate() (interface{}, error) {
	m := r.Model()
	r.setAction(ACTION_CREATE)
	// fill model
	c := r.Context()
	if rb, err := ioutil.ReadAll(c.RequestBody()); err == nil && len(rb) > 0 {
		if err := r.fill(rb); err != nil {
			r.Info("[REST.Init]request body not empty but fill to model failed: %s", err)
		}
	}
	if _, err := m.Valid(); err != nil {
		return nil, err
	}
	return m, nil
}

/* }}} */
/* {{{ func (r *REST) WillCreate() (interface{}, error)
 * React灵感,啥也不做,只等覆盖
 */
func (r *REST) WillCreate() (interface{}, error) {
	return r.Model(), nil
}

/* }}} */
/* {{{ func (r *REST) OnCreate() (interface{}, error)
 *
 */
func (r *REST) OnCreate() (interface{}, error) {
	m := r.Model()
	if r, err := m.CreateRow(); err != nil {
		return nil, err
	} else {
		return r, nil
	}
}

/* }}} */
/* {{{ func (r *REST) DidCreate() (interface{}, error)
 * React灵感,啥也不做,只等覆盖
 */
func (r *REST) DidCreate() (interface{}, error) {
	return r.Model(), nil
}

/* }}} */
/* {{{ func (r *REST) PostCreate() (interface{}, error)
 *
 */
func (r *REST) PostCreate() (interface{}, error) {
	//m := r.SetModel(i.(Model))
	return r.Model().Filter()
}

/* }}} */

/* {{{ func (r *REST) PreUpdate() (interface{}, error)
 *
 */
func (r *REST) PreUpdate() (interface{}, error) {
	m := r.Model()
	r.setAction(ACTION_UPDATE)
	// fill model
	c := r.Context()
	if rb, err := ioutil.ReadAll(c.RequestBody()); err == nil && len(rb) > 0 {
		if err := r.fill(rb); err != nil {
			r.Info("[REST.Init]request body not empty but fill to model failed: %s", err)
		}
	}
	if _, err := m.Valid(); err != nil {
		return nil, err
	}

	return m, nil
}

/* }}} */
/* {{{ func (r *REST) WillUpdate() (interface{}, error)
 *
 */
func (r *REST) WillUpdate() (interface{}, error) {
	return r.Model(), nil
}

/* }}} */
/* {{{ func (r *REST) OnUpdate() (interface{}, error)
 *
 */
func (r *REST) OnUpdate() (interface{}, error) {
	m := r.Model()
	c := r.Context()
	//Info("context: %v", c)
	rk := c.Param(RowkeyKey)
	if affected, err := m.UpdateRow(rk); err != nil {
		return nil, err
	} else {
		if affected <= 0 {
			c.Info("[OnUpdate]not affected any record")
		}
		return m, nil
	}
}

/* }}} */
/* {{{ func (r *REST) DidUpdate() (interface{}, error)
 *
 */
func (r *REST) DidUpdate() (interface{}, error) {
	return r.Model(), nil
}

/* }}} */
/* {{{ func (r *REST) PostUpdate() (interface{}, error)
 *
 */
func (r *REST) PostUpdate() (interface{}, error) {
	//m := r.SetModel(i.(Model))
	return r.Model().Filter()
}

/* }}} */

/* {{{ func (r *REST) PreDelete() (interface{}, error)
 *
 */
func (r *REST) PreDelete() (interface{}, error) {
	return r.Model(), nil
}

/* }}} */
/* {{{ func (r *REST) OnDelete() (interface{}, error)
 *
 */
func (r *REST) OnDelete() (interface{}, error) {
	m := r.Model()
	c := r.Context()
	rk := c.Param(RowkeyKey)
	if affected, err := m.DeleteRow(rk); err != nil {
		return nil, err
	} else {
		if affected <= 0 {
			c.Info("OnDelete not affected any record")
		}
		return m, nil
	}
}

/* }}} */
/* {{{ func (r *REST) PostDelete() (interface{}, error)
 *
 */
func (r *REST) PostDelete() (interface{}, error) {
	return r.Model(), nil
}

/* }}} */

/* {{{ func (r *REST) PreCheck() (interface{}, error)
 *
 */
func (r *REST) PreCheck() (interface{}, error) {
	m := r.Model()
	// 从rcontext里获取条件
	if tr := r.GetEnv(TimeRangeKey); tr != nil { //时间段参数
		m.SetConditions(NewCondition(CTYPE_IS, TAG_TIMERANGE, tr.(*TimeRange)))
	}
	m.SetConditions(r.GetParamConds()...) // 获取参数条件
	return m, nil
}

/* }}} */
/* {{{ func (r *REST) OnCheck() (interface{}, error)
 *
 */
func (r *REST) OnCheck() (interface{}, error) {
	m := r.Model()
	return m.GetCount()
}

/* }}} */
/* {{{ func (r *REST) PostCheck() (interface{}, error)
 *
 */
func (r *REST) PostCheck() (interface{}, error) {
	return r.Model(), nil
}

/* }}} */
