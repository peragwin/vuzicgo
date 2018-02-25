package freqsensor

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/graphql-go/graphql"
)

// Running modes
const (
	NormalMode = iota
	AnimateMode
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
	Mode             int     `json:"mode"`

	WarpOffset float64 `json:"warpOffset"`
	WarpScale  float64 `json:"warpScale"`

	Debug bool `json:"debug"`
}

// Config is passed to initialize the module
type Config struct {
	Buckets    int
	Columns    int
	SampleRate float64
	Parameters *Parameters
}

func (d *FrequencySensor) initGraphql() error {
	paramType, paramMut := NewGraphqlType("ParamType", d.params)

	filterType := graphql.NewObject(
		graphql.ObjectConfig{
			Name: "FilterType",
			Fields: graphql.Fields{
				"amp": &graphql.Field{
					Type: graphql.NewList(graphql.Float),
					Resolve: func(graphql.ResolveParams) (interface{}, error) {
						return d.filterParams.gain.RawMatrix().Data, nil
					},
				},
				"diff": &graphql.Field{
					Type: graphql.NewList(graphql.Float),
					Resolve: func(graphql.ResolveParams) (interface{}, error) {
						return d.filterParams.diff.RawMatrix().Data, nil
					},
				},
			},
		},
	)
	filterMut := &graphql.Field{
		Type: graphql.Boolean,
		Args: graphql.FieldConfigArgument{
			"type":  &graphql.ArgumentConfig{Type: graphql.String},
			"level": &graphql.ArgumentConfig{Type: graphql.Int},
			"gain":  &graphql.ArgumentConfig{Type: graphql.Float},
			"tao":   &graphql.ArgumentConfig{Type: graphql.Float},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			typ, ok := p.Args["type"]
			if !ok {
				return nil, errors.New("missing arg: type")
			}
			level, ok := p.Args["level"]
			if !ok {
				return nil, errors.New("missing arg: level")
			}
			var gain float64
			igain, ok := p.Args["gain"]
			if ok {
				gain = igain.(float64)
			} else {
				gain = 1
			}
			tao, ok := p.Args["tao"]
			if !ok {
				return nil, errors.New("missing arg: tao")
			}
			if err := d.SetFilterParams(
				typ.(string), level.(int), gain, tao.(float64)); err != nil {
				return false, err
			}
			return true, nil
		},
	}

	rootQuery := graphql.NewObject(
		graphql.ObjectConfig{
			Name: "RootQuery",
			Fields: graphql.Fields{
				"params": &graphql.Field{
					Type: paramType,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return d.params, nil
					},
				},
				"filter": &graphql.Field{
					Type: filterType,
					Resolve: func(graphql.ResolveParams) (interface{}, error) {
						return d.filterValues, nil
					},
				},
			},
		},
	)
	rootMut := graphql.NewObject(
		graphql.ObjectConfig{
			Name: "RootMut",
			Fields: graphql.Fields{
				"params": paramMut,
				"filter": filterMut,
			},
		},
	)
	schema, err := graphql.NewSchema(
		graphql.SchemaConfig{
			Query:    rootQuery,
			Mutation: rootMut,
		},
	)
	if err != nil {
		return err
	}
	d.schema = schema
	return nil
}

func (d *FrequencySensor) Query(query string) *graphql.Result {
	return graphql.Do(graphql.Params{
		Schema:        d.schema,
		RequestString: query,
	})
}

// NewGraphqlType expects a pointer type for val
func NewGraphqlType(name string, val interface{}) (*graphql.Object, *graphql.Field) {
	fields := graphql.Fields{}
	mutArgs := graphql.FieldConfigArgument{}

	elem := reflect.ValueOf(val).Elem()
	tagMap := newJSONTagFieldMap(elem)
	ref := elem.Type()

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

	for tag, i := range tagMap {
		if tag == "" {
			continue
		}
		f := ref.Field(i)
		var typ graphql.Type
		switch f.Type.Kind() {
		case reflect.Bool:
			typ = graphql.Boolean
		case reflect.Float32, reflect.Float64:
			typ = graphql.Float
		case reflect.String:
			typ = graphql.String
		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
			typ = graphql.Int
		default:
			panic(fmt.Sprint("unsupported type", f.Type))
		}
		fields[tag] = &graphql.Field{Type: typ, Resolve: resolver(tag)}
		mutArgs[tag] = &graphql.ArgumentConfig{Type: typ}
	}

	return graphql.NewObject(
			graphql.ObjectConfig{
				Name:   name,
				Fields: fields,
			},
		),
		&graphql.Field{
			Type: graphql.Boolean,
			Args: mutArgs,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				for arg, val := range p.Args {
					field := tagMap[arg]
					//ref := reflect.ValueOf(c.Parameters).Elem()
					elem.Field(field).Set(reflect.ValueOf(val))
				}
				return true, nil
			},
		}
}

// NewGraphqlMutationFields expects a pointer type
// func NewGraphqlMutationField(val interface{}) *graphql.Field {
// 	return nil
// }

func getJSONTag(ref reflect.Value, i int) string {
	f := ref.Type().Field(i)
	return jsonTag(&f)
}

func jsonTag(f *reflect.StructField) string {
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
