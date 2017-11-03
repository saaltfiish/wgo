// Package rest provides ...
package rest

import (
	"context"
	"fmt"
	"strings"

	"wgo"

	elastic "gopkg.in/olivere/elastic.v5"
)

var ElasticClient *elastic.Client

func OpenElasticSearch() (err error) {
	ElasticClient, err = elastic.NewClient(
		elastic.SetURL(config.ES["addr"]),
		elastic.SetBasicAuth(config.ES["user"], config.ES["password"]))

	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	var exists bool
	exists, err = ElasticClient.IndexExists(config.ES["index"]).Do(ctx)
	if err != nil {
		panic(err)
	} else if !exists {
		panic("index not exists!")
	}
	return
}

func SearchService() *elastic.SearchService {
	return ElasticClient.Search().Index(config.ES["index"])
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

// es search prepare
func (rest *REST) SearchPrepare() {
	// time range
	qs := make([]elastic.Query, 0)
	c := rest.Context()
	if tr := rest.GetEnv(TimeRangeKey); tr != nil {
		rs := tr.(*TimeRange).Start.UTC().Format("2006-01-02T15:04:05Z07:00")
		re := tr.(*TimeRange).End.UTC().Format("2006-01-02T15:04:05Z07:00")
		Debug("[SearchPrepare]range, start: %s, end: %s", rs, re)
		rangeField := "created"
		if rangeBy := c.QueryParam("range_by"); rangeBy != "" {
			switch strings.ToLower(rangeBy) {
			case "start_time":
				rangeField = "ticket.show.start_time"
			case "created":
			default:
			}
		}
		qs = append(qs, elastic.NewRangeQuery(rangeField).Gte(rs).Lte(re).TimeZone("+08:00"))
	}

	// term
	cons := rest.Conditions()
	if len(cons) > 0 {
	}
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
