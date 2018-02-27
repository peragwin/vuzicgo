package freqsensor

import (
	"testing"
)

// func TestGraphql(t *testing.T) {
// 	c := &Config{
// 		Parameters: DefaultParameters,
// 	}
// 	var err error
// 	c.Schema, err = c.graphql()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	query := `{
// 		params {
// 			gbr
// 			sync
// 		}
// 	}`
// 	res := c.query(query)
// 	if len(res.Errors) > 0 {
// 		t.Fatal(res.Errors)
// 	}

// 	data := struct {
// 		Data struct {
// 			Params Parameters `json:"params"`
// 		} `json:"data"`
// 	}{}
// 	bs, _ := json.Marshal(res)
// 	if err := json.Unmarshal(bs, &data); err != nil {
// 		t.Fatal(err)
// 	}

// 	if data.Data.Params.GlobalBrightness != DefaultParameters.GlobalBrightness {
// 		t.Log(string(bs))
// 		t.Fatal("res value not as expected")
// 	}

// 	mut := `mutation {
// 		params(gbr: 420, sync: 42)
// 	}`
// 	res = c.query(mut)
// 	if len(res.Errors) > 0 {
// 		t.Fatal(res.Errors)
// 	}
// 	// bs, _ = json.Marshal(res)
// 	// t.Error(string(bs))

// 	res = c.query(query)
// 	if len(res.Errors) > 0 {
// 		t.Fatal(res.Errors)
// 	}
// 	bs, _ = json.Marshal(res)
// 	if err := json.Unmarshal(bs, &data); err != nil {
// 		t.Fatal(err)
// 	}

// 	if data.Data.Params.GlobalBrightness != 420 {
// 		t.Log(string(bs))
// 		t.Fatal("gbr not as expected after mut")
// 	}
// 	if data.Data.Params.Sync != 42 {
// 		t.Log(string(bs))
// 		t.Fatal("sync not as expected after mut")
// 	}
// }

func TestNewFields(t *testing.T) {
	typ, mut := NewGraphqlType("params", Parameters{})
	t.Fatal(typ, mut)
}
