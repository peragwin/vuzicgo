import React from 'react';
import { withApollo } from 'react-apollo';

import IconButton from 'material-ui/IconButton';
import IconMenu from 'material-ui/IconMenu';
import MenuItem from 'material-ui/MenuItem';
import AppBar from 'material-ui/AppBar';
import MenuIcon from 'material-ui/svg-icons/navigation/menu'
import CreateIcon from 'material-ui/svg-icons/content/create'
import FolderOpenIcon from 'material-ui/svg-icons/file/folder-open'
import Snackbar from 'material-ui/Snackbar';
import * as Colors from 'material-ui/styles/colors';

import { paramQuery, paramMut } from './ParamController'
import { filterQuery, rawFilterMut } from './FilterController';

const Menu = props =>
  <IconMenu
    iconButtonElement={<IconButton><MenuIcon/></IconButton>}
    targetOrigin={{horizontal: 'left', vertical: 'top'}}
    anchorOrigin={{horizontal: 'left', vertical: 'top'}}
  >
    {["Default", "Bright", "Dim", "Sensitive", "Other"].map(name =>
      <div>
        <MenuItem key={"save"+name} primaryText={`Save Profile: ${name}`}
          leftIcon={<CreateIcon />}
          onClick={() => props.saveProfile(name)} />
        <MenuItem key={"load"+name} primaryText={`Load Profile: ${name}`}
          leftIcon={<FolderOpenIcon />}
          onClick={() => props.loadProfile(name)} />
      </div>)}
  </IconMenu>
Menu.muiName = 'IconMenu'

class appBar extends React.PureComponent {
  state = {
    error: '',
  }

  saveProfile = name => {
    const { client } = this.props
    const { params } = client.readQuery({ query: paramQuery })
    const { filter } = client.readQuery({ query: filterQuery })

    delete params.__typename
    delete filter.__typename

    window.localStorage.setItem(`profile.${name}`, JSON.stringify({params, filter}))
  }

  loadProfile = name => {
    let data = {}
    try {
      data = JSON.parse(window.localStorage.getItem(`profile.${name}`))
    } catch(e) {
      this.setState({error: JSON.stringify(e)})
      return
    }
    if (data === null) {
      this.setState({error: "no saved profile found"})
      return
    }

    const { client } = this.props

    client.mutate({
      mutation: paramMut,
      variables: { params: data.params },
      update: (proxy, {data}) => {
        proxy.writeQuery({ query: paramQuery, data })
      },
    });

    ['amp', 'diff'].forEach(type =>
      client.mutate({
        mutation: rawFilterMut,
        variables: { type, raw: data.filter[type] },
        update: (proxy, {data}) => {
          const cache = client.readQuery({ query: filterQuery })
          const filterData = {...cache.filter}
          filterData[type] = data.rawFilter
          cache.filter = filterData
          proxy.writeQuery({ query: filterQuery, data: cache })
        }
      })
    )
  }
  
  render() {
    const { error } = this.state
    return (
      <div>
        <AppBar
          title="Vizualization Controller"
          titleStyle={{color:'#FFF'}}
          iconElementLeft={
            <Menu saveProfile={this.saveProfile} loadProfile={this.loadProfile}/>}
        />
        <Snackbar
          bodyStyle={{backgroundColor: Colors.redA200}}
          contentStyle={{color: '#FFF'}}
          open={error !== ''}
          message={error}
          autoHideDuration={10000}
        />
      </div>
    )
  }
}

export default withApollo(appBar)