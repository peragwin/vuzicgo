import React, { Component } from 'react';

import './App.css';
import MuiThemeProvider from 'material-ui/styles/MuiThemeProvider';
import getMuiTheme from 'material-ui/styles/getMuiTheme';
import baseTheme from 'material-ui/styles/baseThemes/darkBaseTheme';
import * as Colors from 'material-ui/styles/colors';
import { fade } from 'material-ui/utils/colorManipulator'
import AppBar from 'material-ui/AppBar';

const getTheme = () => {
  let overwrites = {
    "palette": {
        "primary1Color": Colors.deepPurple400
    }
};
  return getMuiTheme(baseTheme, overwrites);
}

const App = () => (
  <MuiThemeProvider muiTheme={getTheme()}>
    <div>
      <AppBar title="Vizualization Controller" />
      <div style={{margin:'2em'}}>
        Main
      </div>
    </div>
  </MuiThemeProvider>
);

export default App;
