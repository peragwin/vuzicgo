package freqsensor

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"

	"gonum.org/v1/gonum/blas/blas64"
	"gonum.org/v1/gonum/mat"

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
	SaturationOffset float64 `json:"satOffset"`
	SaturationScale  float64 `json:"satScale"`
	ValueOffset1     float64 `json:"valueOffset1"`
	ValueOffset2     float64 `json:"valueOffset2"`
	Alpha            float64 `json:"alpha"`
	AlphaOffset      float64 `json:"alphaOffset"`
	AlphaLimit       int     `json:"alphaLimit"`

	Direction        int
	Gain             float64 `json:"gain"`
	DifferentialGain float64 `json:"diff"`
	Preemphasis      float64 `json:"pre"`
	Offset           float64 `json:"offset"`
	Period           int     `json:"period"`
	Sync             float64 `json:"sync"`
	Mode             int     `json:"mode"`

	WarpOffset float64 `json:"warpOffset"`
	WarpScale  float64 `json:"warpScale"`

	Scale       float64 `json:"scale"`
	ScaleOffset float64 `json:"scaleOffset"`

	ColumnDivider int `json:"colDiv"`

	Debug bool `json:"debug"`
}

// Config is passed to initialize the module
type Config struct {
	Buckets    int
	Columns    int
	SampleRate float64
	Parameters *Parameters
}

type saveConfig struct {
	Params  *Parameters      `json:"params"`
	Filters saveFilterValues `json:"filters"`
	Render  *Parameters      `json:"render"`
}

type saveFilterValues struct {
	Gain blas64.General `json:"gain"`
	Diff blas64.General `json:"diff"`
}

func fromFilterValues(fv *filterValues) saveFilterValues {
	return saveFilterValues{
		Gain: fv.gain.RawMatrix(),
		Diff: fv.diff.RawMatrix(),
	}
}

func toFilterValues(sv *saveFilterValues) filterValues {
	return filterValues{
		gain: mat.NewDense(sv.Gain.Rows, sv.Gain.Cols, sv.Gain.Data),
		diff: mat.NewDense(sv.Diff.Rows, sv.Diff.Cols, sv.Diff.Data),
	}
}

// SaveConfig to the given file
func (d *FrequencySensor) SaveConfig(conf string, render *Parameters) error {
	save := saveConfig{
		Params:  d.params,
		Filters: fromFilterValues(&d.filterParams),
		Render:  render,
	}
	fp, err := os.Create(conf)
	if err != nil {
		return err
	}
	defer fp.Close()
	return json.NewEncoder(fp).Encode(&save)
}

// LoadConfig from the given file
func (d *FrequencySensor) LoadConfig(conf string, render *Parameters) error {
	fp, err := os.Open(conf)
	if err != nil {
		if err == os.ErrNotExist {
			return nil
		} else {
			return err
		}
	}
	var save saveConfig
	if err := json.NewDecoder(fp).Decode(&save); err != nil {
		return err
	}
	d.params = save.Params
	d.filterParams = toFilterValues(&save.Filters)
	*render = *save.Render
	return nil
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
		Type: graphql.NewList(graphql.Float),
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
				var fp []float64
				if typ.(string) == "amp" {
					fp = d.filterParams.gain.RawMatrix().Data
				} else if typ.(string) == "diff" {
					fp = d.filterParams.diff.RawMatrix().Data
				}
				gain = math.Abs(fp[2*level.(int)]) + math.Abs(fp[2*level.(int)+1])
			}
			tao, ok := p.Args["tao"]
			if !ok {
				return nil, errors.New("missing arg: tao")
			}
			if err := d.SetFilterParams(
				typ.(string), level.(int), gain, tao.(float64)); err != nil {
				return nil, err
			}
			var ret []float64
			if typ.(string) == "amp" {
				ret = d.filterParams.gain.RawMatrix().Data
			} else if typ.(string) == "diff" {
				ret = d.filterParams.diff.RawMatrix().Data
			}
			return ret, nil
		},
	}
	rawFilterMut := &graphql.Field{
		Type: graphql.NewList(graphql.Float),
		Args: graphql.FieldConfigArgument{
			"type": &graphql.ArgumentConfig{Type: graphql.String},
			"raw":  &graphql.ArgumentConfig{Type: graphql.NewList(graphql.Float)},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			typ, ok := p.Args["type"]
			if !ok {
				return nil, errors.New("missing arg: type")
			}
			r, ok := p.Args["raw"]
			if !ok {
				return nil, errors.New("missing arg: raw")
			}
			rw := r.([]interface{})
			raw := make([]float64, len(rw))
			for i := range rw {
				raw[i] = rw[i].(float64)
			}
			if typ.(string) == "amp" {
				d.filterParams.gain = mat.NewDense(2, 2, raw)
			} else if typ.(string) == "diff" {
				d.filterParams.diff = mat.NewDense(2, 2, raw)
			}
			return raw, nil
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
				"params":    paramMut,
				"filter":    filterMut,
				"rawFilter": rawFilterMut,
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

func (d *FrequencySensor) Query(query string, vars map[string]interface{}) *graphql.Result {
	return graphql.Do(graphql.Params{
		Schema:         d.schema,
		RequestString:  query,
		VariableValues: vars,
	})
}

// NewGraphqlType expects a pointer type for val
func NewGraphqlType(name string, val interface{}) (*graphql.Object, *graphql.Field) {
	fields := graphql.Fields{}
	inputFields := graphql.InputObjectConfigFieldMap{}
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
		inputFields[tag] = &graphql.InputObjectFieldConfig{Type: typ}
		mutArgs[tag] = &graphql.ArgumentConfig{Type: typ}
	}

	paramType := graphql.NewObject(
		graphql.ObjectConfig{
			Name:   name,
			Fields: fields,
		})
	inputParamType := graphql.NewInputObject(
		graphql.InputObjectConfig{
			Name:   "input" + name,
			Fields: inputFields,
		})
	paramMut := &graphql.Field{
		Type: paramType,
		Args: graphql.FieldConfigArgument{
			"params": &graphql.ArgumentConfig{Type: inputParamType},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			params := p.Args["params"].(map[string]interface{})
			for arg, val := range params {
				field := tagMap[arg]
				elem.Field(field).Set(reflect.ValueOf(val))
			}
			return elem.Addr().Interface(), nil
		},
	}

	return paramType, paramMut
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
