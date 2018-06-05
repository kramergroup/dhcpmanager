import React, {Component} from 'react';
import PropTypes from 'prop-types';
import {withStyles} from '@material-ui/core/styles';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import Paper from '@material-ui/core/Paper';
import { ListItemText } from '@material-ui/core';

const style = {
    root: {}
}

const data = [
    "62:e2:45:ee:0b:85",
    "86:d3:9b:08:f9:14",
    "f6:5a:08:8c:d4:50",
    "c2:1c:c3:ec:52:67",
    "e6:1e:8f:b1:f1:3d",
    "f2:ce:e5:9d:97:f5",
    "ba:b6:af:0d:3b:ff",
]

class MacList extends Component {

    render() {

        const {classes} = this.props

        return(
            <Paper className={classes.root}>
                <Table className={classes.table}>
                    <TableHead>
                        <TableRow>
                            <TableCell>MAC</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {data.map( i => {
                            return (
                                <TableRow key={i}>
                                    <TableCell component="th" scope="row">{i}</TableCell>
                                </TableRow>
                            )
                        })
                        }
                    </TableBody>
                </Table>
            </Paper>
        )
    }

}

export default withStyles(style)(MacList)