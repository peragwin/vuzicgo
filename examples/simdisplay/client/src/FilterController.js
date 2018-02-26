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

import { ParamSlider } from './ParamController'

const params = (filterParams, level) => {
  const fp = filterParams.slice(2*level, 2*level+2)
  return {
    tao: tao(fp),
    gain: gain(fp),
  }
}
const tao = filterParams => {
  const g = gain(filterParams)
  if (g === 0) return 1
  return g / filterParams[0]
}
const gain = filterParams => Math.abs(filterParams[0]) + Math.abs(filterParams[1])

const filterQuery = gql`
  query FilterQuery {
    filter {
      amp
      diff
    }
  }`

const filterMut = gql`
  mutation FilterMut ($type: String!, $level: Int!, $gain: Float, $tao: Float!) {
    filter(type: $type, level: $level, gain: $gain, tao: $tao)
  }`

class FilterController extends React.PureComponent {
  state = {
    open: true,
  }

  handleExpand = open => this.setState({ open })

  setFilter = (type, level, attr) => async (e, value) => {
    let gain, tao
    if (attr === 'gain') {
      gain = value
      // we need to extract tao if we are changing gain
      const { filter } = this.props.data
      const filterParams = params(filter[type], level)
      tao = filterParams.tao
    } else if (attr === 'tao') {
      tao = value*value * Math.sign(value)
    }

    const mut = await this.props.setFilter(type, level, tao, gain)
    this.props.update(type, mut)
  }

  render() {
    const { open } = this.state
    const { data: { error, filter } } = this.props
    if (error) console.log(error)
    return (
      <div>
        <Card expanded={open} onExpandChange={this.handleExpand}>
          <CardHeader title="Filter Values"
            subtitle="Make changes to filter values"
            actAsExpander={true}
            showExpandableButton={true}
          />
          <CardText expandable={true}>
            {filter === undefined ? <h3>loading...</h3> :
              <div>
                <p>Edit filter values</p>
                <Divider />

                <h4>Amplitude Filter</h4>
                <h5>Level 0</h5>
                <ParamSlider title="Tao 0"
                  min={-2} max={2}
                  displayValue={params(filter.amp, 0).tao}
                  value={Math.sign(params(filter.amp, 0).tao)*Math.sqrt(Math.abs(params(filter.amp, 0).tao))}
                  onChange={this.setFilter('amp', 0, 'tao')}
                />
                <ParamSlider title="Gain 0"
                  min={-4} max={4}
                  value={params(filter.amp, 0).gain}
                  onChange={this.setFilter('amp', 0, 'gain')}
                />

                <h5>Level 1</h5>
                <ParamSlider title="Tao 1"
                  min={-20} max={20}
                  displayValue={params(filter.amp, 1).tao}
                  value={Math.sign(params(filter.amp, 1).tao)*Math.sqrt(Math.abs(params(filter.amp, 1).tao))}
                  onChange={this.setFilter('amp', 1, 'tao')}
                />
                <ParamSlider title="Gain 1"
                  min={-1} max={1}
                  value={params(filter.amp, 1).gain}
                  onChange={this.setFilter('amp', 1, 'gain')}
                />

                <Divider />

                <h4>Differential Filter</h4>
                <h5>Level 0</h5>
                <ParamSlider title="Tao 0"
                  min={-20} max={20}
                  displayValue={params(filter.diff, 0).tao}
                  value={Math.sign(params(filter.diff, 0).tao)*Math.sqrt(Math.abs(params(filter.diff, 0).tao))}
                  onChange={this.setFilter('diff', 0, 'tao')}
                />
                <ParamSlider title="Gain 0"
                  min={-1} max={1}
                  value={params(filter.diff, 0).gain}
                  onChange={this.setFilter('diff', 0, 'gain')}
                />

                <h5>Level 1</h5>
                <ParamSlider title="Tao 1"
                  min={-20} max={20}
                  displayValue={params(filter.diff, 1).tao}
                  value={Math.sign(params(filter.diff, 1).tao)*Math.sqrt(Math.abs(params(filter.diff, 1).tao))}
                  onChange={this.setFilter('diff', 1, 'tao')}
                />
                <ParamSlider title="Gain 1"
                  min={-1} max={1}
                  value={params(filter.diff, 1).gain}
                  onChange={this.setFilter('diff', 1, 'gain')}
                />
              </div>}
          </CardText>
        </Card>
        <Snackbar
          bodyStyle={{backgroundColor: Colors.redA200}}
          contentStyle={{color: '#FFF'}}
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
  graphql(filterQuery),
  graphql(filterMut, {
    props: ({ mutate, ownProps: {client} }) => ({
      setFilter: (type, level, tao, gain) => mutate({
        mutation: filterMut,
        variables: { type, level, tao, gain },
      }),
      update: (type, { data: {filter} }) => {
        const data = client.readQuery({ query: filterQuery })
        const filterData = {...data.filter}
        filterData[type] = filter
        data.filter = filterData
        client.writeQuery({ query: filterQuery, data })
      },
    })
  }),
)(FilterController)