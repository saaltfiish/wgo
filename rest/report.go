// Package rest provides ...
package rest

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"wgo"
	"wgo/utils"

	"github.com/olivere/elastic"
)

type Report struct {
	Info   ReportInfo  `json:"info,omitempty"`
	Result interface{} `json:"list"`

	base         interface{}
	fields       utils.StructFields
	rest         *REST
	indexName    string
	search       *elastic.SearchService
	limitation   map[string]interface{} // 限制
	params       []string               // 支持参数
	mterms       []string               // 命中的terms查询
	excludes     []string               // source fields exclude
	includes     []string               // source fields include
	dimensions   Worlds                 // 维度
	qs           []elastic.Query        // 根据条件形成的多个query
	filters      map[string]string      // 条件term查询, 需要在nested agg中filter
	size         int                    // 聚合的size
	timestamp    *Timestamp             // 时间戳字段
	timeRange    *TimeRange             // 定义的时间段
	summary      Aggregations           // summary负责统计
	dics         Aggregations           // 字典
	interval     string                 // 时间段 [hour, day, month, year]
	aggregations Aggregations           // 子聚合
	pagination   *Pagination            // 是否可分页
	keywordField string                 // 是否采用多字段模式(主字段text可搜索, 副字段keyword可查询, 这里显示副字段名称)
	id           string
}

type ReportInfo struct {
	Took       int64       `json:"took"`
	Page       int         `json:"page,omitempty"`     //当前页面
	PerPage    int         `json:"per_page,omitempty"` //每页元素个数
	Total      int64       `json:"total"`              // 总数
	Dimensions Worlds      `json:"dimensions,omitempty"`
	Tz         string      `json:"tz,omitempty"`
	Interval   string      `json:"interval,omitempty"`
	Ccy        string      `json:"ccy,omitempty"`
	Start      string      `json:"start,omitempty"`
	End        string      `json:"end,omitempty"`
	First      string      `json:"first,omitempty"` // 第一条数据时间
	Last       string      `json:"last,omitempty"`  // 最后一条数据时间
	Summary    interface{} `json:"summary,omitempty"`
	Dics       interface{} `json:"dics,omitempty"`
}

type Result map[string]interface{}

type Worlds [][]string

// 升维
func (ws Worlds) Increase(worlds ...string) Worlds {
	if ws == nil {
		ws = make([][]string, 0)
	}
	if worlds == nil {
		worlds = []string{}
	}
	ws = append(ws, worlds)
	return ws
}

// 平行
func (ws Worlds) Parallel(world string) Worlds {
	if ws == nil {
		ws = make([][]string, 0)
	}
	d := len(ws)
	ws[d-1] = append(ws[d-1], world)
	return ws
}

type Timestamp struct {
	field string
	min   *time.Time
	max   *time.Time
}

// 聚合
type Aggregation struct {
	field        string
	size         int
	filters      []Filter
	properties   Aggregations // 属性, 即附加字段, 跟field一般为 1:1关系, 一般取最近一个值
	aggregations Aggregations
	wraps        []string
}
type Aggregations []*Aggregation

// reporter
func (rest *REST) Report() *Report {
	if rI := rest.GetEnv(ReportKey); rI != nil {
		if rpt, ok := rI.(*Report); ok {
			return rpt
		}
	}
	return nil
}

// NewReport
func (rest *REST) NewReport(base interface{}, opts ...string) *Report {
	sf := utils.ReadStructFields(base, true, FIELD_TAG, RPT_TAG) // 读json, report两种tag
	fields := utils.ScanStructFields(sf, FIELD_TAG, "", "")
	indexName := es[RCK_REPORTING_INDEX]
	params := []string{}
	if len(opts) > 0 && opts[0] != "" {
		indexName = opts[0]
		params = opts[1:]
	}
	rpt := &Report{
		base:       base,
		fields:     fields,
		rest:       rest,
		indexName:  indexName,
		dimensions: make(Worlds, 0),
		filters:    make(map[string]string),
		search:     SearchService(indexName),
		limitation: map[string]interface{}{
			RTKEY_MIR: float64(200), // interval报表最多200个数据点
		},
	}
	if len(params) > 0 {
		rpt.Params(params...)
	}
	// time range
	if tr := rpt.rest.GetEnv(TimeRangeKey); tr != nil {
		// 传入了时间段参数, 参数优先
		rpt.timeRange = tr.(*TimeRange)
	} else {
		rpt.Today()
	}
	// save to context
	rest.SetEnv(ReportKey, rpt)

	return rpt
}

// NewLogs
func (rest *REST) NewLogs(base interface{}, params ...string) *Report {
	sf := utils.ReadStructFields(base, true, FIELD_TAG, RPT_TAG) // 读json, report两种tag
	fields := utils.ScanStructFields(sf, FIELD_TAG, "", "")
	// for _, f := range fields {
	// 	rest.Info("field: %s, report: %s", f.Tags[FIELD_TAG].Name, f.Tags["report"].Name)
	// }
	rpt := &Report{
		base:       base,
		fields:     fields,
		rest:       rest,
		indexName:  es[RCK_LOGS_INDEX],
		dimensions: make(Worlds, 0),
		filters:    make(map[string]string),
		search:     SearchService(es[RCK_LOGS_INDEX]),
	}
	if len(params) > 0 {
		rpt.Params(params...)
	}
	// time range
	if tr := rpt.rest.GetEnv(TimeRangeKey); tr != nil {
		// 传入了时间段参数, 参数优先
		rpt.timeRange = tr.(*TimeRange)
	}
	return rpt
}

// set index
func (rpt *Report) Search(name string) *Report {
	rpt.indexName = name
	rpt.search = SearchService(name)
	return rpt
}

// keyword field
func (rpt *Report) Size(size int) *Report {
	rpt.size = size
	return rpt
}

// keyword field
func (rpt *Report) KeywordField(kwf string) *Report {
	if kwf != "" {
		rpt.keywordField = kwf
	}
	return rpt
}

// set default time range, 传入一个天数, 代表追溯天数
func (rpt *Report) From(days int) *Report {
	if days > 0 && rpt.timeRange == nil {
		tr := new(TimeRange)
		tr.Start = time.Now().AddDate(0, 0, 0-days).In(wgo.Env().Location)
		tr.End = time.Now().In(wgo.Env().Location)
		rpt.timeRange = tr
	}
	return rpt
}

// set time range to today
func (rpt *Report) Today() *Report {
	// timerange设为当天
	y, m, d := time.Now().In(wgo.Env().Location).Date()
	dtr := new(TimeRange)
	dtr.Start = time.Date(y, m, d, 0, 0, 0, 0, wgo.Env().Location)
	dtr.End = time.Date(y, m, d, 23, 59, 59, 999000000, wgo.Env().Location)
	rpt.timeRange = dtr
	return rpt
}

// params
func (rpt *Report) Params(params ...string) *Report {
	if rpt.params == nil {
		rpt.params = make([]string, 0)
	}
	for _, p := range params {
		// if rpt.rest.Context().QueryParam(p) != "" { // 只有当前端传了这个条件, 才有效
		// 	rpt.params = append(rpt.params, p)
		// }
		rpt.params = append(rpt.params, p)
	}
	return rpt
}

// match term, 作用是相关聚合不需要increase了
func (rpt *Report) MatchTerms(terms ...string) *Report {
	if rpt.mterms == nil {
		rpt.mterms = make([]string, 0)
	}
	for _, t := range terms {
		rpt.mterms = append(rpt.mterms, t)
	}
	return rpt
}

// source exclude
func (rpt *Report) Exclude(fields ...string) *Report {
	if rpt.excludes == nil {
		rpt.excludes = make([]string, 0)
	}
	for _, field := range fields {
		sfn := rpt.SearchFieldName(field)
		// rpt.rest.Debug("exclude field: %s", field, sfn)
		if sfn != "" {
			rpt.excludes = append(rpt.excludes, sfn)
		} else {
			rpt.excludes = append(rpt.excludes, field)
		}
	}
	return rpt
}

// source include
func (rpt *Report) Include(fields ...string) *Report {
	if rpt.includes == nil {
		rpt.includes = make([]string, 0)
	}
	for _, field := range fields {
		sfn := rpt.SearchFieldName(field)
		if sfn != "" {
			rpt.includes = append(rpt.includes, sfn)
		} else {
			rpt.includes = append(rpt.includes, field)
		}
	}
	return rpt
}

// condition
func (rpt *Report) Condition(field string, v ...interface{}) *Report {
	rpt.rest.SetParamConds(NewCondition(CTYPE_IS, field, v...))
	rpt.params = append(rpt.params, field)
	return rpt
}

// summary
func (rpt *Report) Summary(fields ...string) *Report {
	if rpt.summary == nil {
		rpt.summary = make(Aggregations, 0)
	}
	for _, field := range fields {
		agg := NewAggregation(field)
		if rpt.size > 0 {
			agg.Size(rpt.size)
		}
		rpt.summary = append(rpt.summary, agg)
	}
	return rpt
}
func (rpt *Report) AggSummary(agg *Aggregation) *Report {
	if rpt.summary == nil {
		rpt.summary = make(Aggregations, 0)
	}
	rpt.summary = rpt.summary.Push(agg)
	return rpt
}

// field
func (rpt *Report) Field(field string, opts ...string) (sf utils.StructField) {
	if len(rpt.fields) > 0 {
		tag := RPT_TAG
		if len(opts) > 0 {
			tag = opts[0]
		}
		for _, f := range rpt.fields {
			if f.Tags[tag].Name == field {
				return f
			}
		}
	}
	return
}

// 获取完整字段(包含各种嵌套, 以'.'分隔, for es)
func (rpt *Report) SearchFieldName(rf string, opts ...interface{}) string {
	tag := RPT_TAG
	kwf := ""
	if len(opts) > 0 {
		if t, ok := opts[0].(string); ok {
			tag = t
		}
	}
	field := rpt.Field(rf, tag)
	if kwf == "" && field.Tags[RPT_TAG].Options.Contains(RPT_KEYWORD) {
		kwf = RPT_KEYWORD
	} else if len(opts) > 1 {
		if useKWF, ok := opts[1].(bool); ok && useKWF && rpt.keywordField != "" {
			kwf = rpt.keywordField
		}
	}
	if kwf != "" {
		return field.Tags[FIELD_TAG].Name + "." + kwf
	}
	return field.Tags[FIELD_TAG].Name
}

// report type
func reportType(f utils.StructField) string {
	if f.Tags != nil {
		if f.Tags[RPT_TAG].Options.Contains(RPT_SUM) {
			return RPT_SUM
		} else if f.Tags[RPT_TAG].Options.Contains(RPT_TERM) {
			return RPT_TERM
		} else if f.Tags[RPT_TAG].Options.Contains(RPT_RANGE) {
			return RPT_RANGE
		} else if f.Tags[RPT_TAG].Options.Contains(RPT_SEARCH) {
			return RPT_SEARCH
		} else { // 按照类型分
			typ := f.Type
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
			switch typ.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
				return RPT_SUM // 数字默认sum
			default:
				return RPT_TERM
			}
		}
	}
	return ""
}

// NewAggregation
func NewAggregation(field string, properties ...string) *Aggregation {
	agg := &Aggregation{
		field: field,
		wraps: make([]string, 0),
	}
	return agg.Properties(properties...)
}

// properties
func (agg *Aggregation) Properties(properties ...string) *Aggregation {
	if agg.properties == nil {
		agg.properties = make(Aggregations, 0)
	}
	if len(properties) > 0 {
		for _, pf := range properties {
			agg.properties = append(agg.properties, NewAggregation(pf))
		}
	}
	return agg
}

// size
func (agg *Aggregation) Size(size int) *Aggregation {
	agg.size = size
	return agg
}

func (agg *Aggregation) Filters(filters ...Filter) *Aggregation {
	if agg.filters == nil {
		agg.filters = make([]Filter, 0)
	}
	if len(filters) > 0 {
		for _, f := range filters {
			agg.filters = append(agg.filters, f)
		}
	}
	return agg
}

// 增加一个子聚合
func (agg *Aggregation) Sub(sa *Aggregation) *Aggregation {
	if agg.aggregations == nil {
		agg.aggregations = make(Aggregations, 0)
	}
	agg.aggregations = agg.aggregations.Push(sa)
	return agg
}

// 加载最前
func (aggs Aggregations) Unshift(agg *Aggregation) Aggregations {
	aggs = append(Aggregations{agg}, aggs...)
	return aggs
}

// 增加一个平级的聚合
func (aggs Aggregations) Push(agg *Aggregation) Aggregations {
	aggs = append(aggs, agg)
	return aggs
}

func (rpt *Report) NewAggregation(field string, properties ...string) *Aggregation {
	agg := &Aggregation{
		field: field,
	}
	if len(properties) > 0 {
		agg.properties = make(Aggregations, 0)
		for _, pf := range properties {
			agg.properties = append(agg.properties, NewAggregation(pf))
		}
	}
	return agg
}

// timestamp
func (rpt *Report) Timestamp(fields ...string) *Report {
	rpt.timestamp = &Timestamp{field: fields[0]}
	return rpt
}

// monthly, interval = month
func (rpt *Report) Monthly() *Report {
	if rpt.timestamp == nil || rpt.timestamp.field == "" {
		panic("can not interval reporting")
	}
	rpt.Interval(INTVL_MONTH)
	return rpt
}

// weekly, interval = week
func (rpt *Report) Weekly() *Report {
	if rpt.timestamp == nil || rpt.timestamp.field == "" {
		panic("can not interval reporting")
	}
	rpt.Interval(INTVL_WEEK)
	return rpt
}

// daily, interval = day
func (rpt *Report) Daily() *Report {
	if rpt.timestamp == nil || rpt.timestamp.field == "" {
		panic("can not interval reporting")
	}
	rpt.Interval(INTVL_DAY)
	return rpt
}

// hourly, interval = hour
func (rpt *Report) Hourly() *Report {
	if rpt.timestamp == nil || rpt.timestamp.field == "" {
		panic("can not interval reporting")
	}
	rpt.Interval(INTVL_HOUR)
	return rpt
}

// 设置时间段
func (rpt *Report) Interval(inr string) *Report {
	switch inr {
	case INTVL_HOUR, INTVL_DAY, INTVL_WEEK, INTVL_MONTH, INTVL_QUARTER, INTVL_YEAR:
		max := rpt.limitation[RTKEY_MIR].(float64)
		rpt.interval = inr
		// adjust time range,  如果超过max, 则start不变调整end
		hs := rpt.timeRange.End.Sub(rpt.timeRange.Start).Hours()
		count := float64(0)
		switch inr {
		case INTVL_HOUR:
			count = hs
			if count > max {
				rpt.timeRange.End = rpt.timeRange.Start.Add(time.Duration(int32(max/24)*86400-1) * time.Second)
			}
		case INTVL_DAY:
			count = hs / 24
			if count > max {
				rpt.timeRange.End = rpt.timeRange.Start.Add(time.Duration(int32(max)*86400-1) * time.Second)
			}
		case INTVL_WEEK:
			count = hs / (24 * 7)
			if count > max {
				rpt.timeRange.End = rpt.timeRange.Start.Add(time.Duration(int32(max)*7*86400-1) * time.Second)
			}
		case INTVL_MONTH:
			count = hs / (24 * 30)
			if count > max {
				rpt.timeRange.End = rpt.timeRange.Start.Add(time.Duration(int32(max)*30*86400-1) * time.Second)
			}
		case INTVL_QUARTER:
			count = hs / (24 * 30 * 4)
			if count > max {
				rpt.timeRange.End = rpt.timeRange.Start.Add(time.Duration(int32(max)*30*4*86400-1) * time.Second)
			}
		case INTVL_YEAR:
			count = hs / (24 * 365)
			if count > max {
				rpt.timeRange.End = rpt.timeRange.Start.Add(time.Duration(int32(max)*365*86400-1) * time.Second)
			}
		}
	default:
		rpt.rest.Warn("unknown interval: %s", inr)
	}
	return rpt
}

func (rpt *Report) Pagination() *Report {
	rpt.pagination = rpt.rest.Pagination()
	return rpt
}

func (rpt *Report) Id(id string) *Report {
	rpt.id = id
	return rpt
}

// add agg
func (rpt *Report) Aggregation(agg *Aggregation) *Report {
	if rpt.aggregations == nil {
		rpt.aggregations = make(Aggregations, 0)
	}
	rpt.aggregations = rpt.aggregations.Push(agg)
	return rpt
}

// add sum fields
func (rpt *Report) Sum(properties ...string) *Report {
	if rpt.aggregations == nil {
		rpt.aggregations = make(Aggregations, 0)
	}
	for _, pf := range properties {
		agg := NewAggregation(pf)
		rpt.aggregations = rpt.aggregations.Push(agg)
	}
	return rpt
}

// add dic, 利用agg实现字典
func (rpt *Report) Dic(field string, properties ...string) *Report {
	agg := NewAggregation(field, properties...)
	if rpt.size > 0 {
		agg.Size(rpt.size)
	}
	if rpt.dics == nil {
		rpt.dics = make(Aggregations, 0)
	}
	rpt.dics = rpt.dics.Push(agg)
	return rpt
}

// build report
// big
func (rpt *Report) Build() (r Result, err error) {
	tag := RPT_TAG
	// prepare
	if err = rpt.prepare(); err != nil {
		return
	}
	if len(rpt.qs) > 0 {
		rpt.search = rpt.search.Query(elastic.NewBoolQuery().Must(rpt.qs...))
	}

	if rpt.id == "" { // 搜索查询
		// build aggs
		tsField := rpt.SearchFieldName(rpt.timestamp.field)
		// rpt.rest.Info("ts(%s) search field: %s", rpt.timestamp.field, tsField)
		// start/end
		rpt.search = rpt.search.Aggregation(RTKEY_START, MinAgg(tsField)).Aggregation(RTKEY_END, MaxAgg(tsField))

		// aggs
		if rpt.interval != "" {
			// rpt.dimensions = rpt.dimensions.Increase(rpt.interval)
			rpt.dimensions = rpt.dimensions.Increase(RTKEY_TIME)
			tmp := DateHistogramAgg(tsField, rpt.interval)
			if len(rpt.aggregations) > 0 {
				rpt.dimensions = rpt.dimensions.Increase()
				for _, agg := range rpt.aggregations {
					// rpt.rest.Context().Info("agg: %s(%s)", agg.field, rpt.SearchFieldName(agg.field))
					tmp = tmp.SubAggregation(agg.field, rpt.buildAgg(agg, "", true))
				}
			}
			rpt.search = rpt.search.Aggregation(RTKEY_INTVL, tmp)
		} else if len(rpt.aggregations) > 0 {
			rpt.dimensions = rpt.dimensions.Increase()
			for _, agg := range rpt.aggregations {
				rpt.search = rpt.search.Aggregation(agg.field, rpt.buildAgg(agg, "", true))
			}
		}

		// summary
		if len(rpt.summary) > 0 {
			for _, agg := range rpt.summary {
				field := rpt.Field(agg.field, tag)
				switch rtype := reportType(field); rtype {
				case RPT_SUM, RPT_TERM: // 只有sum, term才能做summary
					rpt.rest.Info("summary field: %s", agg.field)
					rpt.search = rpt.search.Aggregation(agg.field, rpt.buildAgg(agg, ""))
				}
			}
		}

		// 字典
		if len(rpt.dics) > 0 {
			for _, agg := range rpt.dics {
				field := rpt.Field(agg.field, tag)
				switch rtype := reportType(field); rtype {
				case RPT_TERM: // 只有term才能做字典
					rpt.search = rpt.search.Aggregation(agg.field, rpt.buildAgg(agg, ""))
				}
			}
		}

		// source filters
		if len(rpt.includes) > 0 {
			rpt.search.FetchSourceContext(elastic.NewFetchSourceContext(true).Include(rpt.includes...))
		}
		if len(rpt.excludes) > 0 {
			rpt.search.FetchSourceContext(elastic.NewFetchSourceContext(true).Exclude(rpt.excludes...))
		}
	}

	//rpt.rest.Info("reporter: %s", rpt)

	// 收获!
	var result *elastic.SearchResult
	result, err = rpt.search.Do(context.Background())
	if err != nil {
		return
	}

	// fetch result
	return rpt.fetch(result)
}

func (rpt *Report) fetch(result *elastic.SearchResult) (r Result, err error) {
	//c := rpt.rest.Context()
	// tz
	rpt.Info.Tz = fmt.Sprint(wgo.Env().Location)
	// ccy
	rpt.Info.Ccy = "CNY"
	// took, hits
	rpt.Info.Took = result.TookInMillis
	// pagination
	if rpt.pagination != nil {
		rpt.Info.Total = result.TotalHits()
		rpt.Info.Page = rpt.pagination.Page
		rpt.Info.PerPage = rpt.pagination.PerPage
	}
	// start/end
	if sv, found := result.Aggregations.MinBucket(RTKEY_START); found && sv.ValueAsString != "" {
		// rpt.rest.Info("end time: %s", sv.ValueAsString)
		st, _ := time.Parse("2006-01-02T15:04:05.000Z07:00", sv.ValueAsString)
		rpt.Info.First = st.In(wgo.Env().Location).Format("2006-01-02T15:04:05Z07:00")
	}
	if ev, found := result.Aggregations.MaxBucket(RTKEY_END); found && ev.ValueAsString != "" {
		// rpt.rest.Info("end time: %s", ev.ValueAsString)
		et, _ := time.Parse("2006-01-02T15:04:05.000Z07:00", ev.ValueAsString)
		rpt.Info.Last = et.In(wgo.Env().Location).Format("2006-01-02T15:04:05Z07:00")
	}
	r = make(Result)

	// summary
	if len(rpt.summary) > 0 {
		summary := make(Result)
		for _, agg := range rpt.summary {
			field := rpt.Field(agg.field, RPT_TAG)
			switch rtype := reportType(field); rtype {
			case RPT_SUM:
				summary[agg.field] = rpt.fetchResult(agg, result.Aggregations)[agg.field]
			case RPT_TERM:
				rst := rpt.fetchResult(agg, result.Aggregations)
				if rst != nil && len(rst) > 0 {
					summary[agg.field] = rst.Results()
				}
			}
		}
		if len(summary) > 0 {
			rpt.Info.Summary = summary
		}
	}

	// 字典
	if len(rpt.dics) > 0 {
		dics := make(Result)
		for _, agg := range rpt.dics {
			field := rpt.Field(agg.field, RPT_TAG)
			switch rtype := reportType(field); rtype {
			case RPT_TERM: // 只有term才能做字典
				rst := rpt.fetchResult(agg, result.Aggregations)
				if rst != nil && len(rst) > 0 {
					dics[agg.field] = rst.Results()
				}
			}
		}
		rpt.Info.Dics = dics
	}

	// hits
	r[RTKEY_HITS] = result.Hits
	// aggs
	if rpt.interval != "" { // interval 先fetch date_histogram数据
		rpt.Info.Interval = rpt.interval
		intvl, found := result.Aggregations.DateHistogram(RTKEY_INTVL)
		if !found {
			err = fmt.Errorf("can't build interval report")
			return
		}
		drts := make([]Result, 0)
		for _, intvlBucket := range intvl.Buckets {
			empty := true
			tr := make(Result)
			tr[RTKEY_TIME] = *intvlBucket.KeyAsString
			if len(rpt.aggregations) > 0 {
				for _, agg := range rpt.aggregations {
					// rpt.rest.Context().Info("agg: %s(%s)", agg.field, rpt.SearchFieldName(agg.field))
					// tr[agg.field] = rpt.fetchResult(agg, intvlBucket.Aggregations)
					field := rpt.Field(agg.field, RPT_TAG)
					switch rtype := reportType(field); rtype {
					case RPT_SUM:
						tr[agg.field] = rpt.fetchResult(agg, intvlBucket.Aggregations)[agg.field]
						if tr[agg.field] != nil && tr[agg.field].(int) > 0 {
							empty = false
						}
					default:
						tr[agg.field] = rpt.fetchResult(agg, intvlBucket.Aggregations)
						if tr[agg.field] != nil && len(tr[agg.field].(Result)) > 0 {
							empty = false
						}
					}
				}
			}
			if !empty {
				drts = append(drts, tr)
			}
		}
		r[RTKEY_INTVL] = Result{RTKEY_RESULTS: drts}
	} else {
		if len(rpt.aggregations) > 0 {
			for _, agg := range rpt.aggregations {
				rst := rpt.fetchResult(agg, result.Aggregations)
				if rst != nil && len(rst) > 0 {
					r[agg.field] = rst
				}
			}
		}
	}
	rpt.Info.Dimensions = rpt.dimensions
	return
}

// 读取report条件
func (rpt *Report) prepare() error {
	r := rpt.rest

	// condition
	if rpt.id != "" {
		rpt.idsQuery()
	} else {
		rpt.search = rpt.search.Size(0)
		// pagination
		if len(rpt.params) > 0 && strings.ToLower(rpt.rest.Context().QueryParam(PARAM_ALLDATA)) == "true" {
			rpt.pagination.Offset = 0
			rpt.pagination.Page = 1
			rpt.pagination.PerPage = 10000
		}
		if rpt.pagination != nil {
			rpt.search = rpt.search.From(rpt.pagination.Offset).Size(rpt.pagination.PerPage)
		} else {
			rpt.search = rpt.search.Size(0)
		}

		rpt.rangeQuery()

		for _, con := range r.GetParamConds() {
			if con.Is != nil && utils.InSliceIgnorecase(con.Field, rpt.params) {
				// 参数的查询条件
				field := rpt.Field(con.Field, RPT_TAG)
				// r.Info("field: %s(%s), type: %s", con.Field, field, reportType(field))
				switch rtype := reportType(field); rtype {
				case RPT_SEARCH:
					rpt.matchPhraseQuery(con.Field, con.Is)
				case RPT_TERM:
					// rpt.rest.Context().Info("condition. field: %s, value: %s", con.Field, con.Is)
					rpt.MatchTerms(con.Field)
					rpt.TermsQuery(con.Field, con.Is)
				}
			}
		}

		// sort
		rpt.search.Sort(rpt.SearchFieldName(rpt.timestamp.field), false)
	}

	return nil
}

// ids query
func (rpt *Report) idsQuery() {
	if rpt.qs == nil {
		rpt.qs = make([]elastic.Query, 0)
	}
	rpt.search = rpt.search.Size(1)
	rpt.qs = append(rpt.qs, elastic.NewIdsQuery(rpt.indexName).Ids(rpt.id))
}

// range query
func (rpt *Report) rangeQuery() {
	if rpt.qs == nil {
		rpt.qs = make([]elastic.Query, 0)
	}
	// 没有查询条件, 或者没有指定all_time=true的时候, 进行时间段查询
	if len(rpt.params) <= 0 || strings.ToLower(rpt.rest.Context().QueryParam(PARAM_ALLTIME)) != "true" {
		// time range
		if rpt.timeRange == nil { // 没传入timerange参数, 也没有设置默认timerange, 当天
			rpt.Today()
		}
		// 如果是interval报表, 则有要求
		rs := rpt.timeRange.Start.In(wgo.Env().Location).Format("2006-01-02T15:04:05Z07:00")
		re := rpt.timeRange.End.In(wgo.Env().Location).Format("2006-01-02T15:04:05Z07:00")
		rpt.Info.Start = rs
		rpt.Info.End = re
		rangeField := rpt.SearchFieldName(rpt.timestamp.field)
		if rangeBy := rpt.rest.Context().QueryParam(PARAM_RANGEBY); rangeBy != "" {
			rangeField = rpt.SearchFieldName(rangeBy)
		}
		//c.Debug("[SearchPrepare]range field: %s, from: %s, to: %s", rangeField, rs, re)
		rpt.qs = append(rpt.qs, elastic.NewRangeQuery(rangeField).Gte(rs).Lte(re).TimeZone(fmt.Sprint(wgo.Env().Location)))
	}
}

// matchPhraseQuery, 如果传入多个参数, 暂时先处理第一个
func (rpt *Report) matchPhraseQuery(field string, text interface{}) {
	if rpt.qs == nil {
		rpt.qs = make([]elastic.Query, 0)
	}
	sfn := rpt.SearchFieldName(field, RPT_TAG)
	switch text.(type) {
	case string:
		rpt.qs = append(rpt.qs, elastic.NewMatchPhraseQuery(sfn, text.(string)))
	case []string:
		if len(text.([]string)) > 0 {
			rpt.qs = append(rpt.qs, elastic.NewMatchPhraseQuery(sfn, text.([]string)[0]))
		}
	}
}

// terms query
func (rpt *Report) TermsQuery(field string, text interface{}) *Report {
	if rpt.qs == nil {
		rpt.qs = make([]elastic.Query, 0)
	}
	f := rpt.Field(field, RPT_TAG)
	sfn := rpt.SearchFieldName(field, RPT_TAG, true)
	// rpt.rest.Info("[TermsQuery] field: %s, text: %#q", field, text)
	v := []interface{}{}
	switch text.(type) {
	case string:
		value := text.(string)
		rpt.filters[field] = value
		v = append(v, value)
	case []string:
		if len(text.([]string)) > 0 {
			for _, t := range text.([]string) {
				v = append(v, t)
			}
		}
	case []interface{}:
		if len(text.([]interface{})) > 0 {
			for _, t := range text.([]interface{}) {
				v = append(v, t)
			}
		}
	}
	if f.Path != "" { // nested
		rpt.qs = append(rpt.qs, elastic.NewNestedQuery(f.Path, elastic.NewTermsQuery(sfn, v...)))
	} else {
		rpt.rest.Info("[TermsQuery], field: %s, value: %#q", sfn, v)
		rpt.qs = append(rpt.qs, elastic.NewTermsQuery(sfn, v...))
	}
	return rpt
}

// build term sagg
func (rpt *Report) buildTermsAgg(agg *Aggregation, path string) (eagg elastic.Aggregation) {
	field := rpt.Field(agg.field, RPT_TAG)
	tsField := rpt.SearchFieldName(rpt.timestamp.field)
	tmp := TermsAgg(rpt.SearchFieldName(agg.field, RPT_TAG, true)).Missing("_empty_")
	if agg.size > 0 {
		tmp.Size(agg.size)
	}
	// properties
	if len(agg.properties) > 0 {
		for _, p := range agg.properties {
			pf := rpt.Field(p.field, RPT_TAG)
			spf := rpt.SearchFieldName(p.field)
			switch rtype := reportType(pf); rtype {
			case RPT_TERM:
				// rpt.rest.Info("term properties: %+v, search field: %s, field: %s", agg.field, spf, pf)
				tmp = tmp.SubAggregation(p.field, LatestField(spf, tsField))
			case RPT_SUM:
				if pf.Path != "" && pf.Path != field.Path { // 父聚合是agg
					tmp = tmp.SubAggregation(p.field, NestedAgg(pf.Path).SubAggregation(p.field, SumAgg(spf)))
					p.wraps = append(p.wraps, RPT_NESTED)
				} else {
					tmp = tmp.SubAggregation(p.field, SumAgg(spf))
				}
			}
		}
	}
	// sub aggs
	if len(agg.aggregations) > 0 {
		rpt.dimensions = rpt.dimensions.Increase()
		for _, subagg := range agg.aggregations {
			tmp = tmp.SubAggregation(subagg.field, rpt.buildAgg(subagg, field.Path, true))
		}
	}
	eagg = tmp
	//当前term在条件查询中, 则先fitler一下
	if v, ok := rpt.filters[agg.field]; ok && v != "" {
		//rpt.rest.Context().Info("add filter. field: %s, value: %s", agg.field, v)
		eagg = FilterAgg(rpt.SearchFieldName(agg.field, RPT_TAG, true), v).SubAggregation(agg.field, eagg)
		agg.wraps = append(agg.wraps, RPT_FILTER)
	}
	// nested agg, 同时在同一nested path下无需nestedagg多次
	if field.Path != "" && field.Path != path {
		//rpt.rest.Context().Info("add nested. path: %s, field: %s", field.Path, agg.field)
		eagg = NestedAgg(field.Path).SubAggregation(agg.field, eagg)
		agg.wraps = append(agg.wraps, RPT_NESTED)
	}
	return
}

// build sum agg
func (rpt *Report) buildSumAgg(agg *Aggregation, path string) (eagg elastic.Aggregation) {
	field := rpt.Field(agg.field, RPT_TAG)
	eagg = SumAgg(rpt.SearchFieldName(agg.field, RPT_TAG))
	// nested agg, 同时在同一nested path下无需nestedagg多次
	if field.Path != "" && field.Path != path {
		//rpt.rest.Context().Info("add nested. path: %s, field: %s", field.Path, agg.field)
		eagg = NestedAgg(field.Path).SubAggregation(agg.field, eagg)
		agg.wraps = append(agg.wraps, RPT_NESTED)
	}
	return
}

// build filters agg
func (rpt *Report) buildFiltersAgg(agg *Aggregation, path string) (eagg elastic.Aggregation) {
	field := rpt.Field(agg.field, RPT_TAG)
	tsField := rpt.SearchFieldName(rpt.timestamp.field)
	tmp := FiltersAgg(agg.filters...)
	// properties
	if len(agg.properties) > 0 {
		for _, p := range agg.properties {
			pf := rpt.Field(p.field, RPT_TAG)
			switch rtype := reportType(pf); rtype {
			case RPT_TERM:
				tmp = tmp.SubAggregation(p.field, LatestField(rpt.SearchFieldName(p.field), tsField))
			case RPT_SUM:
				if pf.Path != "" && pf.Path != field.Path {
					tmp = tmp.SubAggregation(p.field, NestedAgg(pf.Path).SubAggregation(p.field, SumAgg(rpt.SearchFieldName(p.field))))
					p.wraps = append(p.wraps, RPT_NESTED)
				} else {
					tmp = tmp.SubAggregation(p.field, SumAgg(rpt.SearchFieldName(p.field)))
				}
			}
		}
	}
	// sub aggs
	if len(agg.aggregations) > 0 {
		rpt.dimensions = rpt.dimensions.Increase()
		for _, subagg := range agg.aggregations {
			tmp = tmp.SubAggregation(subagg.field, rpt.buildAgg(subagg, field.Path, true))
		}
	}
	eagg = tmp
	//当前term在条件查询中, 则先fitler一下
	if v, ok := rpt.filters[agg.field]; ok && v != "" {
		//rpt.rest.Context().Info("add filter. field: %s, value: %s", agg.field, v)
		eagg = FilterAgg(rpt.SearchFieldName(agg.field, RPT_TAG, true), v).SubAggregation(agg.field, eagg)
		agg.wraps = append(agg.wraps, RPT_FILTER)
	}
	// nested agg, 同时在同一nested path下无需nestedagg多次
	if field.Path != "" && field.Path != path {
		//rpt.rest.Context().Info("add nested. path: %s, field: %s", field.Path, agg.field)
		eagg = NestedAgg(field.Path).SubAggregation(agg.field, eagg)
		agg.wraps = append(agg.wraps, RPT_NESTED)
	}
	return
}

func (rpt *Report) buildAgg(agg *Aggregation, path string, opts ...interface{}) (eagg elastic.Aggregation) {
	field := rpt.Field(agg.field, RPT_TAG)
	switch rtype := reportType(field); rtype {
	case RPT_SUM:
		eagg = rpt.buildSumAgg(agg, path)
	case RPT_TERM:
		if len(opts) > 0 && opts[0] == true {
			//rpt.rest.Context().Info("field: %s", agg.field)
			if !utils.InSliceIgnorecase(agg.field, rpt.mterms) {
				// 聚合字段没有被term查询的时候, 平行维度
				rpt.dimensions = rpt.dimensions.Parallel(agg.field)
			}
		}
		if len(agg.filters) > 0 {
			eagg = rpt.buildFiltersAgg(agg, path)
		} else {
			eagg = rpt.buildTermsAgg(agg, path)
		}
	}
	return
}

func (rpt *Report) fetchResult(agg *Aggregation, aggs elastic.Aggregations) (r Result) {
	field := rpt.Field(agg.field, RPT_TAG)
	// unwrap
	if len(agg.wraps) > 0 {
		for i := len(agg.wraps) - 1; i >= 0; i-- { // reverse walk slice
			//rpt.rest.Context().Info("%s unwrap: %s", agg.field, agg.wraps[i])
			switch wt := agg.wraps[i]; wt {
			case RPT_NESTED:
				if nagg, found := aggs.Nested(agg.field); found {
					//rpt.rest.Context().Info("remove nested. path: %s, field: %s", field.Path, agg.field)
					aggs = nagg.Aggregations
				}
			case RPT_FILTER:
				if v, ok := rpt.filters[agg.field]; ok && v != "" {
					if nagg, found := aggs.Filter(agg.field); found {
						//rpt.rest.Context().Info("remove filter: %s, value: %s", agg.field, v)
						aggs = nagg.Aggregations
					}
				}
			default:
				rpt.rest.Warn("unknown wrap: %s", wt)
			}
		}
	}
	switch rtype := reportType(field); rtype {
	case RPT_SUM:
		if v, found := aggs.Sum(agg.field); found {
			r = Result{agg.field: int(*v.Value)}
		}
	case RPT_TERM:
		if len(agg.filters) > 0 {
			if eagg, found := aggs.Filters(agg.field); found {
				r = make(Result)
				drts := make([]Result, 0)
				for name, bucket := range eagg.NamedBuckets {
					tr := make(Result)
					tr[RTKEY_NAME] = name
					tr[RTKEY_COUNT] = bucket.DocCount
					//tr[agg.field] = bucket.Key.(string)
					// properties
					if len(agg.properties) > 0 {
						for _, p := range agg.properties {
							f := rpt.Field(p.field, RPT_TAG)
							//rpt.rest.Context().Info("fetch property: %s, type: %s, bucket: %v", p.field, reportType(f), bucket)
							rt := rpt.fetchResult(p, bucket.Aggregations)
							switch rtype := reportType(f); rtype {
							case RPT_TERM:
								//tr[p.field] = GetFieldString(bucket, p.field)
								if rts := rt.Results(); len(rts) > 0 {
									tr[p.field] = rts[0][p.field]
								}
							case RPT_SUM:
								tr[p.field] = rt[p.field]
							}
						}
					}
					// sub aggs
					if len(agg.aggregations) > 0 {
						for _, sagg := range agg.aggregations {
							//rpt.rest.Context().Info("sub agg: %s, aggs: %d", sagg.field, len(bucket.Aggregations))
							tr[sagg.field] = rpt.fetchResult(sagg, bucket.Aggregations)
						}
					}
					drts = append(drts, tr)
				}
				r[RTKEY_RESULTS] = drts
			}
		} else {
			if eagg, found := aggs.Terms(agg.field); found && len(eagg.Buckets) > 0 {
				r = make(Result)
				drts := make([]Result, 0)
				for _, bucket := range eagg.Buckets {
					tr := make(Result)
					tr[RTKEY_COUNT] = bucket.DocCount
					tr[agg.field] = bucket.Key.(string)
					// properties
					if len(agg.properties) > 0 {
						for _, p := range agg.properties {
							f := rpt.Field(p.field, RPT_TAG)
							//rpt.rest.Context().Info("fetch property: %s, type: %s, bucket: %v", p.field, reportType(f), bucket)
							rt := rpt.fetchResult(p, bucket.Aggregations)
							switch rtype := reportType(f); rtype {
							case RPT_TERM:
								//tr[p.field] = GetFieldString(bucket, p.field)
								if rts := rt.Results(); len(rts) > 0 {
									tr[p.field] = rts[0][p.field]
								}
							case RPT_SUM:
								tr[p.field] = rt[p.field]
							}
						}
					}
					// sub aggs
					if len(agg.aggregations) > 0 {
						for _, sagg := range agg.aggregations {
							//rpt.rest.Context().Info("sub agg: %s, aggs: %d", sagg.field, len(bucket.Aggregations))
							tr[sagg.field] = rpt.fetchResult(sagg, bucket.Aggregations)
						}
					}
					drts = append(drts, tr)
				}
				r[RTKEY_RESULTS] = drts
			}
		}
	}
	return
}

// 提取系列
func (r Result) IntervalResults() []Result {
	if intvl, ok := r[RTKEY_INTVL]; ok {
		if dr, ok := intvl.(Result); ok {
			return dr.Results()
		}
	}
	return nil
}
func (r Result) Results() []Result {
	if r == nil {
		return nil
	}
	if rs, ok := r[RTKEY_RESULTS]; ok {
		if rts, ok := rs.([]Result); ok {
			return rts
		}
	}
	return nil
}
func Results(i interface{}) []Result {
	if r, ok := i.(Result); ok {
		return r.Results()
	}
	return nil
}

func (r Result) Property(field string) interface{} {
	if k, ok := r[field]; ok {
		return k
	}
	return nil
}

func (r Result) Count() int64 {
	if k, ok := r[RTKEY_COUNT]; ok {
		return k.(int64)
	}
	return 0
}
func (r Result) Name() string {
	if k, ok := r[RTKEY_NAME]; ok {
		return k.(string)
	}
	return ""
}
func (r Result) Hits() *elastic.SearchHits {
	if hits := r.Property(RTKEY_HITS); hits != nil {
		return hits.(*elastic.SearchHits)
	}
	return nil
}

// 自动根据传入对象的结构, 解析数据, 略牛
func (r Result) ExtractTo(ob interface{}, ifs ...utils.StructField) (interface{}, bool) {
	// Info("results: %+v", r)
	found := false
	var fs utils.StructFields
	if len(ifs) > 0 {
		fs = utils.StructFields(ifs)
	} else {
		fs = utils.ReadStructFields(ob, true, FIELD_TAG, RPT_TAG)
	}
	if fs != nil {
		for _, f := range fs {
			if len(f.SubFields) <= 0 { // 尝试解析赋值
				// Info("report field: %s", f.Tags[RPT_TAG].Name)
				if p := r.Property(f.Tags[RPT_TAG].Name); p != nil {
					switch v := p.(type) {
					case string:
						// Info("%s is string, value: %s", f.Tags[RPT_TAG].Name, v)
						ob = utils.Instance(ob)
						fv := utils.FieldByIndex(reflect.ValueOf(ob), f.Index)
						switch f.Type.String() {
						case "string":
							fv.Set(reflect.ValueOf(v))
						case "*string":
							fv.Set(reflect.ValueOf(&v))
						default:
							Error("%s's type not match int: %s", f.Tags[RPT_TAG].Name, f.Type.String())
						}
					case int:
						// Info("%s is int, value: %d", f.Tags[RPT_TAG].Name, v)
						ob = utils.Instance(ob)
						fv := utils.FieldByIndex(reflect.ValueOf(ob), f.Index)
						switch f.Type.String() {
						case "int64":
							fv.Set(reflect.ValueOf(int64(v)))
						case "*int64":
							vv := int64(v)
							fv.Set(reflect.ValueOf(&vv))
						case "int":
							fv.Set(reflect.ValueOf(v))
						case "*int":
							fv.Set(reflect.ValueOf(&v))
						default:
							Error("%s's type not match int: %s", f.Tags[RPT_TAG].Name, f.Type.String())
						}
					default:
						Error("%s unknown type, value: %+v", v)
					}
					found = true
				}
			} else { // struct, 继续解析
				// Info("struct field: %+v, %+v", f.Type, f.Tags[RPT_TAG].Name, f.SubFields)
				ob = utils.Instance(ob)
				fv := utils.FieldByIndex(reflect.ValueOf(ob), f.Index)
				// fv.Set(reflect.ValueOf(utils.Instance(reflect.Indirect(reflect.New(fv.Type())).Interface())))
				if fv.Type().Kind() == reflect.Ptr {
					fv.Set(reflect.ValueOf(utils.Instance(reflect.Indirect(reflect.New(fv.Type())).Interface())))
				} else {
					fv.Set(reflect.ValueOf(utils.Instance(reflect.Indirect(reflect.New(fv.Type())).Interface())).Elem())
				}
				if _, f := r.ExtractTo(ob, f.SubFields...); !f { //  如果没有找到, 置为空值
					fv.Set(reflect.Zero(fv.Type()))
				} else {
					found = true
				}
			}
		}
	} else {
		Info("extrac erro: %+v", ob)
	}
	// Info("extrac: %+v", ob)
	return ob, found
}
