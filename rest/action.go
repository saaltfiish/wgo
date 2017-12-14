package rest

type Action interface { // 把action定义为一个interface, 这样业务代码可以overwrite
	PreGet(i interface{}) (interface{}, error)  //获取前
	OnGet(i interface{}) (interface{}, error)   //获取中
	PostGet(i interface{}) (interface{}, error) //获取后

	PreSearch(i interface{}) (interface{}, error)  // 搜索前
	OnSearch(i interface{}) (interface{}, error)   // 搜索
	PostSearch(i interface{}) (interface{}, error) // 搜索后

	PreCreate(i interface{}) (interface{}, error)  // 创建前
	WillCreate(i interface{}) (interface{}, error) // 创建hook
	OnCreate(i interface{}) (interface{}, error)   // 创建中
	DidCreate(i interface{}) (interface{}, error)  // 创建hook
	PostCreate(i interface{}) (interface{}, error) // 创建后

	PreUpdate(i interface{}) (interface{}, error)  // 更新前
	OnUpdate(i interface{}) (interface{}, error)   // 更新中
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

/* {{{ func (rest *REST) Trigger(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) Trigger(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
/* {{{ func (rest *REST) Defer(i interface{})
 *
 */
func (rest *REST) Defer(i interface{}) {
}

/* }}} */

/* {{{ func (rest *REST) PreGet(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) PreGet(i interface{}) (interface{}, error) {
	m := i.(Model)
	c := rest.Context()
	// pk,放入条件
	if id := c.Param(RowkeyKey); id != "" {
		pk, _, _ := m.PKey()
		//c.Debug("[PreGet][pk: %s, id: %s]", pk, id)
		m.SetConditions(NewCondition(CTYPE_IS, pk, id))
	}
	// 从restcontext里获取条件
	if tr := rest.GetEnv(TimeRangeKey); tr != nil { //时间段参数
		m.SetConditions(NewCondition(CTYPE_IS, TAG_TIMERANGE, tr.(*TimeRange)))
	}
	if cons := rest.GetEnv(ConditionsKey); cons != nil { //从context里面获取参数条件
		m.SetConditions(cons.([]*Condition)...)
	}
	// fields
	if fs := rest.GetEnv(FieldsKey); fs != nil { //从context里面获取参数条件
		m.SetFields(fs.([]string)...)
	}
	return i, nil
}

/* }}} */
/* {{{ func (rest *REST) OnGet(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) OnGet(i interface{}) (interface{}, error) {
	m := i.(Model)
	if row, err := m.Row(m); err == nil {
		return rest.SetModel(row.(Model)), err
	} else {
		return nil, err
	}
}

/* }}} */
/* {{{ func (rest *REST) PostGet(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) PostGet(i interface{}) (interface{}, error) {
	if i != nil {
		if m, ok := i.(Model); ok {
			return m.Protect()
		}
	}
	return i, nil
}

/* }}} */

/* {{{ func (rest *REST) PreSearch(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) PreSearch(i interface{}) (interface{}, error) {
	m := i.(Model)
	// 从rest里获取条件
	if p := rest.GetEnv(PaginationKey); p != nil { //排序
		m.SetPagination(p.(*Pagination))
	}
	if ob := rest.GetEnv(OrderByKey); ob != nil { //排序
		m.SetConditions(NewCondition(CTYPE_ORDER, TAG_ORDERBY, ob.(*OrderBy)))
	}
	if tr := rest.GetEnv(TimeRangeKey); tr != nil { //时间段参数
		m.SetConditions(NewCondition(CTYPE_RANGE, TAG_TIMERANGE, tr.(*TimeRange)))
	}
	if cons := rest.GetEnv(ConditionsKey); cons != nil { //从context里面获取参数条件
		m.SetConditions(cons.([]*Condition)...)
	}
	// fields
	if fs := rest.GetEnv(FieldsKey); fs != nil { //从context里面获取参数条件
		m.SetFields(fs.([]string)...)
	}
	return i, nil
}

/* }}} */
/* {{{ func (rest *REST) OnSearch(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) OnSearch(i interface{}) (interface{}, error) {
	m := i.(Model)
	return m.Rows()
}

/* }}} */
/* {{{ func (rest *REST) PostSearch(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) PostSearch(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */

/* {{{ func (rest *REST) PreCreate(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) PreCreate(i interface{}) (interface{}, error) {
	m := i.(Model)
	var err error
	if m, err = m.Valid(); err != nil {
		return nil, err
	}
	return m, nil
}

/* }}} */
/* {{{ func (rest *REST) WillCreate(i interface{}) (interface{}, error)
 * React灵感,啥也不做,只等覆盖
 */
func (rest *REST) WillCreate(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
/* {{{ func (rest *REST) OnCreate(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) OnCreate(i interface{}) (interface{}, error) {
	m := i.(Model)
	if r, err := m.CreateRow(); err != nil {
		return nil, err
	} else {
		rest.SetModel(r)
		return r, nil
	}
}

/* }}} */
/* {{{ func (rest *REST) DidCreate(i interface{}) (interface{}, error)
 * React灵感,啥也不做,只等覆盖
 */
func (rest *REST) DidCreate(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
/* {{{ func (rest *REST) PostCreate(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) PostCreate(i interface{}) (interface{}, error) {
	//m := rest.SetModel(i.(Model))
	return i.(Model).Filter()
}

/* }}} */

/* {{{ func (rest *REST) PreUpdate(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) PreUpdate(i interface{}) (interface{}, error) {
	m := i.(Model)
	var err error
	if m, err = m.Valid(); err != nil {
		return nil, err
	}

	return m, nil
}

/* }}} */
/* {{{ func (rest *REST) OnUpdate(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) OnUpdate(i interface{}) (interface{}, error) {
	m := i.(Model)
	c := rest.Context()
	//Info("context: %v", c)
	rk := c.Param(RowkeyKey)
	if affected, err := m.UpdateRow(rk); err != nil {
		return nil, err
	} else {
		if affected <= 0 {
			c.Info("OnUpdate not affected any record")
		}
		return m, nil
	}
}

/* }}} */
/* {{{ func (rest *REST) PostUpdate(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) PostUpdate(i interface{}) (interface{}, error) {
	//m := rest.SetModel(i.(Model))
	return i.(Model).Filter()
}

/* }}} */

/* {{{ func (rest *REST) PreDelete(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) PreDelete(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
/* {{{ func (rest *REST) OnDelete(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) OnDelete(i interface{}) (interface{}, error) {
	m := i.(Model)
	c := rest.Context()
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
/* {{{ func (rest *REST) PostDelete(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) PostDelete(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */

/* {{{ func (rest *REST) PreCheck(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) PreCheck(i interface{}) (interface{}, error) {
	m := i.(Model)
	// 从restcontext里获取条件
	if tr := rest.GetEnv(TimeRangeKey); tr != nil { //时间段参数
		m.SetConditions(NewCondition(CTYPE_IS, TAG_TIMERANGE, tr.(*TimeRange)))
	}
	if cons := rest.GetEnv(ConditionsKey); cons != nil { //从context里面获取参数条件
		m.SetConditions(cons.([]*Condition)...)
	}
	return i, nil
}

/* }}} */
/* {{{ func (rest *REST) OnCheck(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) OnCheck(i interface{}) (interface{}, error) {
	m := i.(Model)
	return m.GetCount()
}

/* }}} */
/* {{{ func (rest *REST) PostCheck(i interface{}) (interface{}, error)
 *
 */
func (rest *REST) PostCheck(i interface{}) (interface{}, error) {
	return i, nil
}

/* }}} */
