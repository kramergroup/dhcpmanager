import React from 'react';
import TableCell from '@material-ui/core/TableCell';
import {withStyles} from '@material-ui/core/styles';

// Status: 0-Unbound, 1-Bound, 2-Stale, 3-Stopped
const colors = ['#ffdb4d','#009900','#990000','#e6e6e6'];
const stroke = 3;

const styles = {
  icon: {
    transform: 'translate(0,3px)',
  },
};

class StatusTableCell extends React.Component {

  render() {

    const { classes } = this.props;

    var w = parseInt(this.props.size) + stroke;
    var h = parseInt(this.props.size) + stroke;
    var r = parseInt(this.props.size) / 2;
    var c = colors[this.props.status];

    return (
      <TableCell className={this.props.className}>
        <svg width={w} height={h} className={classes.icon}>
            <circle cx={w/2} cy={h/2} r={r} stroke={c} fill='transparent' stroke-width={stroke}/>
        </svg> 
      </TableCell>
    );

  }

};

export default withStyles(styles)(StatusTableCell);