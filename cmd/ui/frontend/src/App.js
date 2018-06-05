import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
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
  }
};

class App extends Component {

  title = "Network Interfaces"

  render() {

    const { classes } = this.props;

    return (
      <div className={classes.root}>
        <TopBar></TopBar>
        <div className={classes.content}>
          <div className={classes.deviceTable}>
            <DeviceTable endpoint="ws://localhost:8080/ws/allocations"></DeviceTable>
          </div>
          <div className={classes.macList}>
            <MacPlot endpoint="ws://localhost:8080/ws/macpool" width="300" height="300"></MacPlot>
          </div>
        </div>
      </div>
    );
  }
}

App.propTypes = {
  classes: PropTypes.object.isRequired,
};

export default withStyles(styles)(App);
