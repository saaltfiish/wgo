// Package rest provides ...
package rest

import (
	"context"
	"fmt"
	"reflect"
	"wgo/utils"

	"wgo"

	elastic "github.com/olivere/elastic/v7"
)

var ElasticClient *elastic.Client

func OpenElasticSearch() (err error) {
	if wgo.Env().DebugMode {
		ElasticClient, err = elastic.NewClient(
			elastic.SetURL(es[RCK_ES_ADDR]),
			elastic.SetSniff(false),
			elastic.SetBasicAuth(es[RCK_ES_USER], es[RCK_ES_PWD]),
			elastic.SetTraceLog(logger),
		)
	} else {
		ElasticClient, err = elastic.NewClient(
			elastic.SetURL(es[RCK_ES_ADDR]),
			elastic.SetSniff(false),
			elastic.SetBasicAuth(es[RCK_ES_USER], es[RCK_ES_PWD]),
		)
	}

	if err != nil {
		wgo.Error("[OpenElasticSearch]error: %s", err)
		return err
	}
	// ctx := context.Background()
	// var exists bool
	// exists, err = ElasticClient.IndexExists(es[RCK_REPORTING_INDEX]).Do(ctx)
	// if err != nil {
	// 	panic(err)
	// } else if !exists {
	// 	panic(fmt.Sprintf("reporting index(%s) not exists!", es[RCK_REPORTING_INDEX]))
	// }
	// exists, err = ElasticClient.IndexExists(es[RCK_LOGS_INDEX]).Do(ctx)
	// if err != nil {
	// 	panic(err)
	// } else if !exists {
	// 	panic(fmt.Sprintf("logs index(%s) not exists!", es[RCK_LOGS_INDEX]))
	// }
	// micro services
	return
}

func SearchService(index string) *elastic.SearchService {
	return ElasticClient.Search().Index(index)
}

func BulkService(index string) *elastic.BulkService {
	return ElasticClient.Bulk().Index(index)
}

func NewBulkIndexRequest() *elastic.BulkIndexRequest {
	return elastic.NewBulkIndexRequest()
}

// get field
func GetFieldString(bucket *elastic.AggregationBucketKeyItem, f string) string {
	if aggs, found := bucket.Aggregations.Terms(f); found {
		if len(aggs.Buckets) > 0 {
			return aggs.Buckets[0].Key.(string)
		}
	}
	return ""
}

func GetResultSum(result *elastic.SearchResult, f string) int {
	if v, found := result.Aggregations.Sum(f); found {
		return int(*v.Value)
	}
	return 0
}

func GetFieldSum(bucket *elastic.AggregationBucketKeyItem, f string) int {
	if v, found := bucket.Aggregations.Sum(f); found {
		return int(*v.Value)
	}
	return 0
}

type Filter struct {
	Name   string
	Field  string
	Values []interface{}
}

// aggs
func TermsAgg(field string) *elastic.TermsAggregation {
	return elastic.NewTermsAggregation().Field(field).Size(1000)
}
func TopAgg(field string, ascending bool) *elastic.TopHitsAggregation {
	return elastic.NewTopHitsAggregation().Sort(field, ascending).Size(1)
}
func ReverseAgg() *elastic.ReverseNestedAggregation {
	return elastic.NewReverseNestedAggregation()
}
func SumAgg(field string) *elastic.SumAggregation {
	return elastic.NewSumAggregation().Field(field)
}
func CumSumAgg(field string) *elastic.CumulativeSumAggregation {
	return elastic.NewCumulativeSumAggregation().BucketsPath(field)
}
func NestedAgg(field string) *elastic.NestedAggregation {
	return elastic.NewNestedAggregation().Path(field)
}
func MinAgg(field string) *elastic.MinAggregation {
	return elastic.NewMinAggregation().Field(field)
}
func MaxAgg(field string) *elastic.MaxAggregation {
	return elastic.NewMaxAggregation().Field(field)
}
func AvgAgg(field string) *elastic.AvgAggregation {
	return elastic.NewAvgAggregation().Field(field)
}
func DateHistogramAgg(field, interval string) *elastic.DateHistogramAggregation {
	// return elastic.NewDateHistogramAggregation().Field(field).Interval(interval).TimeZone(fmt.Sprint(wgo.Env().Location))
	switch interval {
	case INTVL_HOUR, INTVL_DAY, INTVL_WEEK, INTVL_MONTH, INTVL_QUARTER, INTVL_YEAR:
		return elastic.NewDateHistogramAggregation().Field(field).CalendarInterval(interval).TimeZone(fmt.Sprint(wgo.Env().Location))
	default:
		return elastic.NewDateHistogramAggregation().Field(field).FixedInterval(interval).TimeZone(fmt.Sprint(wgo.Env().Location))
	}
}
func FilterAgg(field string, value ...interface{}) *elastic.FilterAggregation {
	return elastic.NewFilterAggregation().Filter(elastic.NewTermsQuery(field, value...))
}
func FiltersAgg(filters ...Filter) *elastic.FiltersAggregation {
	fa := elastic.NewFiltersAggregation()
	for _, f := range filters {
		fa = fa.FilterWithName(f.Name, elastic.NewTermsQuery(f.Field, f.Values...))
	}
	return fa
}
func NestedTermsAgg(path, field string) *elastic.NestedAggregation {
	return NestedAgg(path).SubAggregation(field, TermsAgg(field))
}

// 获取某字段最近的一个值, 通过子聚合排序, tf代表时间戳字段
func LatestField(f, tf string) *elastic.TermsAggregation {
	return TermsAgg(f).Size(1).OrderByAggregation("_ts_", false).SubAggregation("_ts_", MaxAgg(tf))
}

// 获取某个字段所有值
func Fields(f string) *elastic.TermsAggregation {
	return TermsAgg(f)
}

// save to es
func saveToES(m Model) {
	defer func() {
		if err := recover(); err != nil {
			Error("error: %s", err)
		}
	}()
	if _, pk, _ := m.PKey(); pk != "" {
		idx := fmt.Sprintf("%s%s", esPrefix, m.TableName())
		exists, _ := ElasticClient.IndexExists(idx).Do(context.Background())
		if !exists {
			// create index
			ElasticClient.CreateIndex(idx)
		}
		nm, _ := m.Row()
		bulk := ElasticClient.Bulk().Index(idx)
		bulk.Add(NewBulkIndexRequest().Id(pk).Doc(nm))
		bulk.Do(context.Background())
	} else {
		Warn("not found primary key")
	}
}

// save all rows to es
func SaveAllToES(m Model) {
	defer func() {
		if err := recover(); err != nil {
			Error("error: %s", err)
		}
	}()
	idx := fmt.Sprintf("%s%s", esPrefix, m.TableName())
	exists, _ := ElasticClient.IndexExists(idx).Do(context.Background())
	if !exists {
		// create index
		ElasticClient.CreateIndex(idx)
	}
	bulk := ElasticClient.Bulk().Index(idx)
	if rs, err := m.Rows(); err == nil {
		rsv := reflect.ValueOf(rs)
		for i := 0; i < rsv.Len(); i++ {
			mi := rsv.Index(i).Interface()
			if _, pk, _ := primaryKey(mi.(Model)); pk != "" {
				bulk.Add(NewBulkIndexRequest().Id(utils.MustString(pk)).Doc(mi))
			}
		}
		if cnt := bulk.NumberOfActions(); cnt > 0 {
			_, be := bulk.Do(context.Background())
			if be != nil {
				wgo.Warn("bulk error: %s", be)
			}
		}
	}
}
