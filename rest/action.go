package rest

type Action interface { // 把action定义为一个interface, 这样业务代码可以overwrite
	PreGet(i interface{}) (interface{}, error)  //获取前
	WillGet(i interface{}) (interface{}, error) //获取前hook
	OnGet(i interface{}) (interface{}, error)   //获取中
	DidGet(i interface{}) (interface{}, error)  //获取到hook
	PostGet(i interface{}) (interface{}, error) //获取后

	PreSearch(i interface{}) (interface{}, error)  // 搜索前
	WillSearch(i interface{}) (interface{}, error) // 搜索前hook
	OnSearch(i interface{}) (interface{}, error)   // 搜索
	DidSearch(i interface{}) (interface{}, error)  // 搜索完hook
	PostSearch(i interface{}) (interface{}, error) // 搜索后

	PreCreate(i interface{}) (interface{}, error)  // 创建前
	WillCreate(i interface{}) (interface{}, error) // 创建hook
	OnCreate(i interface{}) (interface{}, error)   // 创建中
	DidCreate(i interface{}) (interface{}, error)  // 创建hook
	PostCreate(i interface{}) (interface{}, error) // 创建后

	PreUpdate(i interface{}) (interface{}, error)  // 更新前
	WillUpdate(i interface{}) (interface{}, error) // hook
	OnUpdate(i interface{}) (interface{}, error)   // 更新中
	DidUpdate(i interface{}) (interface{}, error)  // hook
	PostUpdate(i interface{}) (interface{}, error) // 更新后

	PreDelete(i interface{}) (interface{}, error)  // 删除前
	OnDelete(i interface{}) (interface{}, error)   // 删除中
	PostDelete(i interface{}) (interface{}, error) // 删除后

	PreCheck(i interface{}) (interface{}, error)  // 搜索前
	OnCheck(i interface{}) (interface{}, error)   // 搜索中
	PostCheck(i interface{}) (interface{}, error) // 搜索后

	Trigger(i interface{}) (interface{}, error) //触发器
	Defer(i interface{})
}

/* {{{ func (r *REST) Trigger(i interface{}) (interface{}, error)
 *
 */
func (r *REST) Trigger(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
/* {{{ func (r *REST) Defer(i interface{})
 *
 */
func (r *REST) Defer(i interface{}) {
}

/* }}} */

/* {{{ func (r *REST) PreGet(i interface{}) (interface{}, error)
 *
 */
func (r *REST) PreGet(i interface{}) (interface{}, error) {
	m := i.(Model)
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
	return i, nil
}

/* }}} */
/* {{{ func (r *REST) WillGet(i interface{}) (interface{}, error)
 * React灵感,啥也不做,只等覆盖
 */
func (r *REST) WillGet(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
/* {{{ func (r *REST) OnGet(i interface{}) (interface{}, error)
 *
 */
func (r *REST) OnGet(i interface{}) (interface{}, error) {
	m := i.(Model)
	if row, err := m.Row(); err == nil {
		return r.SetModel(row.(Model)), err
		// return row, err
	} else {
		return nil, err
	}
}

/* }}} */
/* {{{ func (r *REST) DidGet(i interface{}) (interface{}, error)
 * React灵感,啥也不做,只等覆盖
 */
func (r *REST) DidGet(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
/* {{{ func (r *REST) PostGet(i interface{}) (interface{}, error)
 *
 */
func (r *REST) PostGet(i interface{}) (interface{}, error) {
	if i != nil {
		if m, ok := i.(Model); ok {
			return m.Protect()
		}
	}
	return i, nil
}

/* }}} */

/* {{{ func (r *REST) PreSearch(i interface{}) (interface{}, error)
 *
 */
func (r *REST) PreSearch(i interface{}) (interface{}, error) {
	m := i.(Model)
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
	return i, nil
}

/* }}} */
/* {{{ func (r *REST) WillSearch(i interface{}) (interface{}, error)
 * React灵感,啥也不做,只等覆盖
 */
func (r *REST) WillSearch(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
/* {{{ func (r *REST) OnSearch(i interface{}) (interface{}, error)
 *
 */
func (r *REST) OnSearch(i interface{}) (interface{}, error) {
	m := i.(Model)
	return m.List()
}

/* }}} */
/* {{{ func (r *REST) DidSearch(i interface{}) (interface{}, error)
 * React灵感,啥也不做,只等覆盖
 */
func (r *REST) DidSearch(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
/* {{{ func (r *REST) PostSearch(i interface{}) (interface{}, error)
 *
 */
func (r *REST) PostSearch(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */

/* {{{ func (r *REST) PreCreate(i interface{}) (interface{}, error)
 *
 */
func (r *REST) PreCreate(i interface{}) (interface{}, error) {
	m := i.(Model)
	var err error
	if m, err = m.Valid(); err != nil {
		return nil, err
	}
	return m, nil
}

/* }}} */
/* {{{ func (r *REST) WillCreate(i interface{}) (interface{}, error)
 * React灵感,啥也不做,只等覆盖
 */
func (r *REST) WillCreate(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
/* {{{ func (r *REST) OnCreate(i interface{}) (interface{}, error)
 *
 */
func (r *REST) OnCreate(i interface{}) (interface{}, error) {
	m := i.(Model)
	if r, err := m.CreateRow(); err != nil {
		return nil, err
	} else {
		// r.SetModel(r)
		return r, nil
	}
}

/* }}} */
/* {{{ func (r *REST) DidCreate(i interface{}) (interface{}, error)
 * React灵感,啥也不做,只等覆盖
 */
func (r *REST) DidCreate(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
/* {{{ func (r *REST) PostCreate(i interface{}) (interface{}, error)
 *
 */
func (r *REST) PostCreate(i interface{}) (interface{}, error) {
	//m := r.SetModel(i.(Model))
	return i.(Model).Filter()
}

/* }}} */

/* {{{ func (r *REST) PreUpdate(i interface{}) (interface{}, error)
 *
 */
func (r *REST) PreUpdate(i interface{}) (interface{}, error) {
	m := i.(Model)
	var err error
	if m, err = m.Valid(); err != nil {
		return nil, err
	}

	return m, nil
}

/* }}} */
/* {{{ func (r *REST) WillUpdate(i interface{}) (interface{}, error)
 *
 */
func (r *REST) WillUpdate(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
/* {{{ func (r *REST) OnUpdate(i interface{}) (interface{}, error)
 *
 */
func (r *REST) OnUpdate(i interface{}) (interface{}, error) {
	m := i.(Model)
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
/* {{{ func (r *REST) DidUpdate(i interface{}) (interface{}, error)
 *
 */
func (r *REST) DidUpdate(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
/* {{{ func (r *REST) PostUpdate(i interface{}) (interface{}, error)
 *
 */
func (r *REST) PostUpdate(i interface{}) (interface{}, error) {
	//m := r.SetModel(i.(Model))
	return i.(Model).Filter()
}

/* }}} */

/* {{{ func (r *REST) PreDelete(i interface{}) (interface{}, error)
 *
 */
func (r *REST) PreDelete(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
/* {{{ func (r *REST) OnDelete(i interface{}) (interface{}, error)
 *
 */
func (r *REST) OnDelete(i interface{}) (interface{}, error) {
	m := i.(Model)
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
/* {{{ func (r *REST) PostDelete(i interface{}) (interface{}, error)
 *
 */
func (r *REST) PostDelete(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */

/* {{{ func (r *REST) PreCheck(i interface{}) (interface{}, error)
 *
 */
func (r *REST) PreCheck(i interface{}) (interface{}, error) {
	m := i.(Model)
	// 从rcontext里获取条件
	if tr := r.GetEnv(TimeRangeKey); tr != nil { //时间段参数
		m.SetConditions(NewCondition(CTYPE_IS, TAG_TIMERANGE, tr.(*TimeRange)))
	}
	m.SetConditions(r.GetParamConds()...) // 获取参数条件
	return i, nil
}

/* }}} */
/* {{{ func (r *REST) OnCheck(i interface{}) (interface{}, error)
 *
 */
func (r *REST) OnCheck(i interface{}) (interface{}, error) {
	m := i.(Model)
	return m.GetCount()
}

/* }}} */
/* {{{ func (r *REST) PostCheck(i interface{}) (interface{}, error)
 *
 */
func (r *REST) PostCheck(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
