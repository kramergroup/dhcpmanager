import React, {Component} from 'react';
import PropTypes from 'prop-types';
import {withStyles} from '@material-ui/core/styles';
import Typography from '@material-ui/core/Typography';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import Paper from '@material-ui/core/Paper';
import Websocket from 'react-websocket';

const styles = {}

let id = 0;

function createData(name, calories, fat, carbs, protein) {
  id += 1;
  return {id, name, calories, fat, carbs, protein};
}

const data = [
  createData('Frozen yoghurt', 159, 6.0, 24, 4.0),
  createData('Ice cream sandwich', 237, 9.0, 37, 4.3),
  createData('Eclair', 262, 16.0, 24, 6.0),
  createData('Cupcake', 305, 3.7, 67, 4.3),
  createData('Gingerbread', 356, 16.0, 49, 3.9),
];


class DeviceTable extends Component {

  constructor(props) {
    super(props);
    this.state = {
      data: []
    };
  }

  handleData(data) {
    if (data != null && data != "") {
      let result = JSON.parse(data);
      this.setState({data:result.Data});
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
                  <TableCell>{n.Lease !== null ? n.Lease.Expire : "n/a"}</TableCell>
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