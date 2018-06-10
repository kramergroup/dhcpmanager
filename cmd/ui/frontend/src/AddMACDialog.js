import React from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import Button from '@material-ui/core/Button';
import Dialog from '@material-ui/core/Dialog';
import Paper from '@material-ui/core/Paper';
import TextField from '@material-ui/core/TextField';
import AppBar from '@material-ui/core/AppBar';
import Toolbar from '@material-ui/core/Toolbar';
import IconButton from '@material-ui/core/IconButton';
import Typography from '@material-ui/core/Typography';
import CloseIcon from '@material-ui/icons/Close';
import Slide from '@material-ui/core/Slide';

const styles = {
  appBar: {
    position: 'relative',
  },
  flex: {
    flex: 1,
  },
  container: {
    padding: '2em',
  },
  paper: {
    padding: '2em',
  },
  textField: {
    padding: '0.4em',
    border: 'solid 1px lightblue',
    width: '100%',
  },
};

function Transition(props) {
  return <Slide direction="up" {...props} />;
}

class AddMACDialog extends React.Component {

  state = {
    macs: []
  }

  handleSave = () => {
    fetch(this.props.endpoint, {
      method: 'POST',

      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        macs: this.state.macs,
      }),
    })
    .then( response => {
      if (response.status >= 200 && response.status < 300) {
          return response.json();
      } else {
        var error = new Error(response.statusText);
        error.response = response;
        throw error;
      }
    })
    .then( response => {
      if (response.status !== 'success') {
        var error = new Error(response.info);
        error.status = response.status;
        throw error;
      }
      // Success - just close the window
      this.props.onClose()
    }
    )
    .catch( error => {
      alert("There was an error adding MACS. ["+error+"]");
    });
  }

  handleChange = event => {
    var text = event.target.value;
    var lines = text.split('\n');
    this.setState({macs: lines});
  }

  render() {
    const { classes } = this.props;
    const open = this.props.open;
    const onClose = this.props.onClose;
    const handleSave = this.handleSave;
    return (
      <div>
        <Dialog fullScreen
                open={open}
                onClose={onClose}
                TransitionComponent={Transition}>
          <AppBar className={classes.appBar}>
            <Toolbar>
              <IconButton color="inherit" onClick={onClose} aria-label="Close">
                <CloseIcon />
              </IconButton>
              <Typography variant="title" color="inherit" className={classes.flex}>
                Add Media Access Control Addresses
              </Typography>
              <Button color="inherit" onClick={handleSave}>
                Save
              </Button>
            </Toolbar>
          </AppBar>
          <div className={classes.container}>
          <Paper className={classes.paper}>
            <Typography variant="title" color="inherit" className={classes.flex}>
            Please enter valid MAC addresses, one per line 
            </Typography>
            <TextField id="multiline-mac" 
                       multiline rows="30"
                       placeholder="ab:34:ef:12:34:aa"
                       className={classes.textField}
                       onChange={this.handleChange}
                       margin="normal"/>
          </Paper>
          </div>
        </Dialog>
      </div>
    );
  }
}

AddMACDialog.propTypes = {
  classes: PropTypes.object.isRequired,
};

export default withStyles(styles)(AddMACDialog);