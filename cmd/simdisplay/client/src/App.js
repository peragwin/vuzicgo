import React from 'react';

import { ApolloProvider } from 'react-apollo';
import { ApolloClient } from 'apollo-client';
import { HttpLink } from 'apollo-link-http';
import { InMemoryCache } from 'apollo-cache-inmemory';

import './App.css';
import MuiThemeProvider from 'material-ui/styles/MuiThemeProvider';
import getMuiTheme from 'material-ui/styles/getMuiTheme';
import baseTheme from 'material-ui/styles/baseThemes/darkBaseTheme';
import * as Colors from 'material-ui/styles/colors';
// import { fade } from 'material-ui/utils/colorManipulator'

import AppBar from './AppBar'
import ParamController from './ParamController'
import FilterController from './FilterController'

const client = new ApolloClient({
  // By default, this client will send queries to the
  //  `/graphql` endpoint on the same host
  link: new HttpLink({ uri: '/api/v2/graphql' }),
  cache: new InMemoryCache()
});

const getTheme = () => {
  let overwrites = {
    "palette": {
        "primary1Color": Colors.purple600
    }
  };
  return getMuiTheme(baseTheme, overwrites);
}

const App = () => (
  <ApolloProvider client={client}>
    <MuiThemeProvider muiTheme={getTheme()}>
      <div>
        <AppBar />
        <div style={{margin:'2em'}}>
          <ParamController />
          <FilterController />
        </div>
      </div>
    </MuiThemeProvider>
  </ApolloProvider>
);

export default App;
