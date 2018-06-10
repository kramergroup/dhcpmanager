import React from 'react';
import TableCell from '@material-ui/core/TableCell';

// Status: 0-Unbound, 1-Bound, 2-Stale, 3-Stopped
const colors = ['yellow','green','red','grey'];

class StatusTableCell extends React.Component {

  render() {

    var w = this.props.size;
    var h = this.props.size;
    var status = this.props.status;

    return (
      <TableCell>
        <svg width={w} height={h}>
            <circle cx={w/2} cy={h/2} stroke={colors[status]} stroke-width="3px"/>
        </svg> 
      </TableCell>
    );

  }

}

export default StatusTableCell;