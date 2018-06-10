import React from 'react';
import TableCell from '@material-ui/core/TableCell';

// Status: 0-Unbound, 1-Bound, 2-Stale, 3-Stopped
const colors = ['#ffdb4d','#009900','#990000','#e6e6e6'];
const stroke = 3;

class StatusTableCell extends React.Component {

  render() {

    var w = parseInt(this.props.size) + stroke;
    var h = parseInt(this.props.size) + stroke;
    var r = parseInt(this.props.size) / 2;
    var c = colors[this.props.status];

    return (
      <TableCell className={this.props.className}>
        <svg width={w} height={h}>
            <circle cx={w/2} cy={h/2} r={r} stroke={c} fill='transparent' stroke-width={stroke}/>
        </svg> 
      </TableCell>
    );

  }

}

export default StatusTableCell;