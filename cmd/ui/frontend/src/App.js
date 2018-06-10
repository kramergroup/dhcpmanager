import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import Typography from '@material-ui/core/Typography';

import { MuiThemeProvider, createMuiTheme } from '@material-ui/core/styles';
import blue from '@material-ui/core/colors/blue';

import DeviceTable from './DeviceTable'
import TopBar from './TopBar'
import MacPlot from './MacPlot'
import AddMACDialog from './AddMACDialog'

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

const theme = createMuiTheme({
  palette: {
    primary: blue,
  },
  typography: {
    fontFamily: ['Lato','Roboto','sans-serif'],
  },
});

var getLocation = function(href) {
  var l = document.createElement("a");
  l.href = href;
  return l;
};


class App extends Component {

  title = "Network Interfaces"

  state = {
    showMACDialog: false,
  }

  wsUrl(s) {
    var l = window.location;
    if ( process.env.REACT_APP_ENDPOINT !== '') {
      l = getLocation(process.env.REACT_APP_ENDPOINT);
    }
    return ((l.protocol === "https:") ? "wss://" : "ws://") + l.host + l.pathname + s;
  }

  apiUrl(s) {
    var l = window.location;
    if ( process.env.REACT_APP_ENDPOINT !== '') {
      l = getLocation(process.env.REACT_APP_ENDPOINT);
    }
    return l.protocol + "//" + l.host + l.pathname + s;
  }

  handleAddClick = () => {
    this.setState({showMACDialog: true});
  }

  closeMACDialog = () => {
    this.setState({showMACDialog: false});
  }

  render() {

    const { classes } = this.props;

    return (
      <MuiThemeProvider theme={theme}>
      <div className={classes.root}>
        <TopBar onAddClick={this.handleAddClick}></TopBar>
        <div className={classes.content}>
          <div className={classes.deviceTable}>
            <Typography variant="headline" component="h2">
             Allocations
            </Typography>
            <DeviceTable endpoint={this.wsUrl("ws/allocations")}></DeviceTable>
          </div>
          <div className={classes.macList}>
            <Typography variant="headline" component="h2">
              Device Address Pool
            </Typography>
            <MacPlot endpoint={this.wsUrl("ws/macpool")} width="300" height="300"></MacPlot>
          </div>
        </div>
        <AddMACDialog open={this.state.showMACDialog} 
                      onClose={this.closeMACDialog}
                      endpoint={this.apiUrl("api/macs")}>
        </AddMACDialog>
      </div>
      </MuiThemeProvider>
    );
  }
}

App.propTypes = {
  classes: PropTypes.object.isRequired,
};

export default withStyles(styles)(App);
