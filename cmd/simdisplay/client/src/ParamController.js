import React from 'react'

import { graphql, compose, withApollo } from 'react-apollo';
import gql from 'graphql-tag';

import {
    Card, CardHeader, CardText,
} from 'material-ui/Card'
import Slider from 'material-ui/Slider'
import Divider from 'material-ui/Divider';
import { Snackbar } from 'material-ui';

import * as Colors from 'material-ui/styles/colors';

/*
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
*/

export const paramQuery = gql`
  query ParamQuery {
    params {
      gbr
      period
      br
      satOffset
      valueOffset1
      valueOffset2
      alpha
      alphaOffset
      alphaLimit
      gain
      pre
      offset
      diff
      sync
      warpScale
      warpOffset
      scale
    }
  }`

export const paramMut = gql`
  mutation ParamMut ($params: inputParamType!) {
    params(params: $params) {
      gbr
      period
      br
      gain
      pre
      satOffset
      valueOffset1
      valueOffset2 
      alpha
      alphaOffset
      alphaLimit
      offset
      diff
      sync
      warpScale
      warpOffset
      scale
    }
  }
`

const sliderStyle = {
    width: '90%',
    margin: 'auto',
}

export const ParamSlider = ({ title, min, max, value, onChange, step, displayValue }) =>
    <div>
        <h5>{title}</h5>
        <span>{(displayValue || value) === undefined ? 'not set' : (displayValue || value)}</span>
        <Slider style={sliderStyle}
            min={min} max={max}
            step={step}
            value={value || 0}
            onChange={onChange}
        />
    </div>

class ParamEditor extends React.PureComponent {
    state = {
        open: true,
    }

    handleExpand = open => this.setState({ open })

    setParam = param => async (e, value) => {
        const mut = await this.props.setParam(param, value)
        // const { data: { params } } = mut
        this.props.update(mut)
    }

    render() {
        const { open } = this.state
        let { data: { error, params } } = this.props
        if (error) console.log(error)
        params = params || {}
        return (
            <div>
                <Card expanded={open} onExpandChange={this.handleExpand}>
                    <CardHeader
                        title="Parameter Editor"
                        subtitle="Make changes to display parameters"
                        actAsExpander={true}
                        showExpandableButton={true}
                    />
                    <CardText expandable={true}>
                        <p>Edit stuff here</p>

                        <Divider />

                        <ParamSlider title="Global Brightness"
                            min={0} max={255} step={1}
                            value={params.gbr}
                            onChange={this.setParam('gbr')}
                        />
                        <ParamSlider title="Period"
                            min={1} max={256} step={1}
                            value={params.period}
                            onChange={this.setParam('period')}
                        />

                        <Divider />

                        <ParamSlider title="Saturation"
                            min={0} max={8}
                            value={params.br}
                            onChange={this.setParam('br')}
                        />
                        <ParamSlider title="Saturation Offset"
                            min={-4} max={4}
                            value={params.satOffset}
                            onChange={this.setParam('satOffset')}
                        />
                        <ParamSlider title="Intensity"
                            min={0} max={8}
                            value={params.gain}
                            onChange={this.setParam('gain')}
                        />
                        <ParamSlider title="Intensity Offset"
                            min={0} max={8}
                            value={params.offset}
                            onChange={this.setParam('offset')}
                        />
                        <ParamSlider title="Value Offset 1"
                            min={-2} max={2}
                            value={params.valueOffset1}
                            onChange={this.setParam('valueOffset1')}
                        />
                        <ParamSlider title="Value Offset 2"
                            min={-6} max={6}
                            value={params.valueOffset2}
                            onChange={this.setParam('valueOffset2')}
                        />
                        <ParamSlider title="Alpha"
                            min={0} max={2}
                            value={params.alpha}
                            onChange={this.setParam('alpha')}
                        />
                        <ParamSlider title="Alpha Offset"
                            min={-6} max={6}
                            value={params.alphaOffset}
                            onChange={this.setParam('alphaOffset')}
                        />
                        <ParamSlider title="Differential Intensity"
                            min={0} max={0.05} step={0.0002}
                            value={params.diff}
                            onChange={this.setParam('diff')}
                        />
                        <ParamSlider title="Preemphasis"
                            min={1} max={16}
                            value={params.pre}
                            onChange={this.setParam('pre')}
                        />

                        <Divider />

                        <ParamSlider title="Warp Intensity"
                            min={0} max={4}
                            value={params.warpScale}
                            onChange={this.setParam('warpScale')}
                        />
                        <ParamSlider title="Warp Offset"
                            min={0} max={2}
                            value={params.warpOffset}
                            onChange={this.setParam('warpOffset')}
                        />
                        <ParamSlider title="Scale"
                            min={0} max={4}
                            value={params.scale}
                            onChange={this.setParam('scale')}
                        />
                        <ParamSlider title="Scale Offset"
                            min={0} max={2}
                            value={params.scaleOffset}
                            onChange={this.setParam('scaleOffset')}
                        />

                        <Divider />

                        <ParamSlider title="Color Sync Force"
                            min={0} max={0.05} step={0.0002}
                            value={params.sync}
                            onChange={this.setParam('sync')}
                        />

                    </CardText>
                </Card>
                <Snackbar
                    bodyStyle={{ backgroundColor: Colors.redA200 }}
                    contentStyle={{ color: '#FFF' }}
                    open={error !== undefined}
                    message={error ? error.message : ''}
                    autoHideDuration={10000}
                />
            </div>
        )
    }
}

export default compose(
    withApollo,
    graphql(paramQuery),
    graphql(paramMut, {
        props: ({ mutate, ownProps: { client } }) => ({
            setParam: (param, value) => mutate({
                mutation: paramMut,
                variables: { params: { [param]: value } }
            }),
            update: ({ data: { params } }) => {
                // Read the data from our cache for this query.
                const data = client.readQuery({ query: paramQuery });

                // Add our todo from the mutation.
                params = { ...data.params, ...params }
                data.params = params

                // Write our data back to the cache.
                client.writeQuery({ query: paramQuery, data });
            },
        }),
    }),
)(ParamEditor)