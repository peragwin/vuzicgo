package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/graphql-go/graphql"
)

// Parameters is a set of parameters that control the visualization
type Parameters struct {
	GlobalBrightness float64 `json:"gbr"`
	Brightness       float64 `json:"br"`
	Direction        int
	Gain             float64 `json:"gain"`
	DifferentialGain float64 `json:"diff"`
	Offset           float64 `json:"offset"`
	Period           int     `json:"period"`
	Sync             float64 `json:"sync"`
}

// Config is passed to initialize the module
type Config struct {
	Buckets    int
	Columns    int
	SampleRate float64
	Parameters *Parameters

	schema graphql.Schema
}

// NewConfig inits a config with a corresponding graphql schema
func NewConfig(cfg *Config) *Config {
	var err error
	cfg.schema, err = cfg.graphql()
	if err != nil {
		panic(err)
	}
	return cfg
}

func (c *Config) graphql() (graphql.Schema, error) {

	tagMap := newJSONTagFieldMap(reflect.ValueOf(Parameters{}))

	resolver := func(tag string) func(graphql.ResolveParams) (interface{}, error) {
		field, ok := tagMap[tag]
		if !ok {
			panic("unknown tag: " + tag)
		}
		return func(p graphql.ResolveParams) (interface{}, error) {
			if params, ok := p.Source.(*Parameters); ok {
				ref := reflect.ValueOf(params).Elem()
				val := ref.Field(field)
				return val.Interface(), nil
			}
			return nil, fmt.Errorf("something when wrong: %#v", p.Source)
		}
	}

	paramType := graphql.NewObject(
		graphql.ObjectConfig{
			Name: "ParamType",
			Fields: graphql.Fields{
				"gbr": &graphql.Field{
					Type:    graphql.Float,
					Resolve: resolver("gbr"),
				},
				"br": &graphql.Field{
					Type:    graphql.Float,
					Resolve: resolver("br"),
				},
				"gain": &graphql.Field{
					Type:    graphql.Float,
					Resolve: resolver("gain"),
				},
				"diff": &graphql.Field{
					Type:    graphql.Float,
					Resolve: resolver("diff"),
				},
				"offset": &graphql.Field{
					Type:    graphql.Float,
					Resolve: resolver("offset"),
				},
				"period": &graphql.Field{
					Type:    graphql.Int,
					Resolve: resolver("period"),
				},
				"sync": &graphql.Field{
					Type:    graphql.Float,
					Resolve: resolver("sync"),
				},
			},
		},
	)

	rootQuery := graphql.NewObject(
		graphql.ObjectConfig{
			Name: "RootQuery",
			Fields: graphql.Fields{
				"params": &graphql.Field{
					Type: paramType,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return c.Parameters, nil
					},
				},
			},
		},
	)
	rootMut := graphql.NewObject(
		graphql.ObjectConfig{
			Name: "RootMut",
			Fields: graphql.Fields{
				"params": &graphql.Field{
					Type: graphql.Float,
					Args: graphql.FieldConfigArgument{
						"gbr":    &graphql.ArgumentConfig{Type: graphql.Float},
						"br":     &graphql.ArgumentConfig{Type: graphql.Float},
						"gain":   &graphql.ArgumentConfig{Type: graphql.Float},
						"diff":   &graphql.ArgumentConfig{Type: graphql.Float},
						"offset": &graphql.ArgumentConfig{Type: graphql.Float},
						"period": &graphql.ArgumentConfig{Type: graphql.Int},
						"sync":   &graphql.ArgumentConfig{Type: graphql.Float},
					},
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						for arg, val := range p.Args {
							field := tagMap[arg]
							ref := reflect.ValueOf(c.Parameters).Elem()
							ref.Field(field).Set(reflect.ValueOf(val))
						}
						return nil, nil
					},
				},
			},
		},
	)
	return graphql.NewSchema(
		graphql.SchemaConfig{
			Query:    rootQuery,
			Mutation: rootMut,
		},
	)
}

func (c *Config) query(query string) *graphql.Result {
	return graphql.Do(graphql.Params{
		Schema:        c.schema,
		RequestString: query,
	})
}

func getJSONTag(ref reflect.Value, i int) string {
	f := ref.Type().Field(i)
	t := f.Tag.Get("json")
	return strings.Split(t, ",")[0]
}

func newJSONTagFieldMap(ref reflect.Value) map[string]int {
	m := make(map[string]int)
	for i := 0; i < ref.NumField(); i++ {
		tag := getJSONTag(ref, i)
		m[tag] = i
	}
	return m
}
