import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import Typography from '@material-ui/core/Typography';

import { MuiThemeProvider, createMuiTheme } from '@material-ui/core/styles';
import blue from '@material-ui/core/colors/blue';

import DeviceTable from './DeviceTable'
import TopBar from './TopBar'
import MacPlot from './MacPlot'

import './App.css';

const styles = {
  root: {
    flexGrow: 2,
  },
  content: {
    display: 'flex',
  },
  deviceTable: {
    flex: 3,
    margin: '2em',
    marginRight: '1em',
  },
  macList: {
    flex: 1,
    width: '0%',
    margin: '2em',
    marginLeft: '1em',
    textAlign: 'center',
  }
};

const font = "'Lato', sans-serif"; 

const theme = createMuiTheme({
  palette: {
    primary: blue,
  },
  typography: {
    fontFamily: ['Lato','Roboto','sans-serif'],
  },
});


class App extends Component {

  title = "Network Interfaces"

  url(s) {
    var l = window.location;
    return ((l.protocol === "https:") ? "wss://" : "ws://") + l.host + l.pathname + s;
  }

  render() {

    const { classes } = this.props;

    return (
      <MuiThemeProvider theme={theme}>
      <div className={classes.root}>
        <TopBar></TopBar>
        <div className={classes.content}>
          <div className={classes.deviceTable}>
            <Typography variant="headline" component="h2">
             Allocations
            </Typography>
            <DeviceTable endpoint={this.url("ws/allocations")}></DeviceTable>
          </div>
          <div className={classes.macList}>
            <Typography variant="headline" component="h2">
              Device Address Pool
            </Typography>
            <MacPlot endpoint={this.url("ws/macpool")} width="300" height="300"></MacPlot>
          </div>
        </div>
      </div>
      </MuiThemeProvider>
    );
  }
}

App.propTypes = {
  classes: PropTypes.object.isRequired,
};

export default withStyles(styles)(App);
