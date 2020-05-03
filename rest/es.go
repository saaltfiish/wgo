// Package rest provides ...
package rest

import (
	"context"
	"fmt"

	"wgo"

	elastic "github.com/olivere/elastic/v7"
)

var ElasticClient *elastic.Client

func OpenElasticSearch() (err error) {
	ElasticClient, err = elastic.NewClient(
		elastic.SetURL(es[RCK_ES_ADDR]),
		elastic.SetSniff(false),
		elastic.SetBasicAuth(es[RCK_ES_USER], es[RCK_ES_PWD]))

	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	var exists bool
	exists, err = ElasticClient.IndexExists(es[RCK_REPORTING_INDEX]).Do(ctx)
	if err != nil {
		panic(err)
	} else if !exists {
		panic(fmt.Sprintf("reporting index(%s) not exists!", es[RCK_REPORTING_INDEX]))
	}
	exists, err = ElasticClient.IndexExists(es[RCK_LOGS_INDEX]).Do(ctx)
	if err != nil {
		panic(err)
	} else if !exists {
		panic(fmt.Sprintf("logs index(%s) not exists!", es[RCK_LOGS_INDEX]))
	}
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
func ReverseAgg() *elastic.ReverseNestedAggregation {
	return elastic.NewReverseNestedAggregation()
}
func SumAgg(field string) *elastic.SumAggregation {
	return elastic.NewSumAggregation().Field(field)
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
func DateHistogramAgg(field, interval string) *elastic.DateHistogramAggregation {
	return elastic.NewDateHistogramAggregation().Field(field).Interval(interval).TimeZone(fmt.Sprint(wgo.Env().Location))
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
