import React, {Component} from 'react';
import PropTypes from 'prop-types';
import {withStyles} from '@material-ui/core/styles';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import Paper from '@material-ui/core/Paper';
import Websocket from 'react-websocket';

const styles = {
  root: {
    marginTop: '16px'
  }
}

class DeviceTable extends Component {

  constructor(props) {
    super(props);
    this.state = {
      data: []
    };
  }

  handleData(data) {
    if (data !== null && data !== "") {
      let result = JSON.parse(data);
      this.setState({data:result.Data});
    }
  }

  formateTime(dateString) {
    var date = new Date(dateString);
    var seconds = Math.floor((new Date() - date) / 1000);
    var interval = Math.floor(seconds / 86400);
    if (interval > 1) {
      return "+" + interval + " d " + date.getHours() + ":" + date.getMinutes();
    } else {
      return date.getHours() + ":" + date.getMinutes();
    }

  }

  render() {

    const {classes} = this.props;

    return (
        <Paper className={classes.root}>
          <Table className={classes.table}>
            <TableHead>
              <TableRow>
                <TableCell>Hostname</TableCell>
                <TableCell>IP</TableCell>
                <TableCell>MAC</TableCell>
                <TableCell>Expires</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {this.state.data.map(n => {
                return (
                  <TableRow key={n.id}>
                    <TableCell component="th" scope="row">{n.Hostname}</TableCell> 
                    <TableCell>{n.Lease !== null ? n.Lease.FixedAddress : "n/a"}</TableCell>
                    <TableCell>{n.Interface.HardwareAddr}</TableCell>
                    <TableCell>{n.Lease !== null ? this.formateTime(n.Lease.Expire) : "n/a"}</TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
          <Websocket url={this.props.endpoint}
              onMessage={this.handleData.bind(this)}/>
        </Paper>
    );
  }

}


DeviceTable.propTypes = {
  classes: PropTypes.object.isRequired,
};


export default withStyles(styles)(DeviceTable);