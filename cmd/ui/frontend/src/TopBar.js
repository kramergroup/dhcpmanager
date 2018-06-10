import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import AppBar from '@material-ui/core/AppBar';
import Toolbar from '@material-ui/core/Toolbar';
import Typography from '@material-ui/core/Typography';
import Button from '@material-ui/core/Button';

const styles = {
  flex: {
    flex: 1,
  },
};

class TopBar extends Component {

  render() {

    const {classes} = this.props;

    return (
      <AppBar position="static">
        <Toolbar>
          <Typography variant="title" color="inherit" className={classes.flex}>
            Network Interfaces
          </Typography>
          <Button color="inherit" onClick={this.props.onAddClick}>Add</Button>
        </Toolbar>
      </AppBar>
    )

  }

}

TopBar.propTypes = {
  classes: PropTypes.object.isRequired,
};

export default withStyles(styles)(TopBar)
