import React, {Component} from 'react';
import {withStyles} from '@material-ui/core/styles';
import {arc} from 'd3-shape'
import {pie} from 'd3-shape'
import {scaleLinear} from 'd3-scale'
import Websocket from 'react-websocket';

const style = theme => ({
  root: theme.mixins.gutters({
    paddingTop: 16,
    paddingBottom: 16,
    textAlign: 'center',
  })
});

class MacPlot extends Component {

  width=300
  height=300

  constructor(props) {
    super(props)
    this.state = {data:[0,0]}
  }

  handleData(update) {
    let result = JSON.parse(update);
    this.setState({data: [result.bound, result.available]})
  }

  render() {
    
    const {classes} = this.props

    var radius = Math.min(this.props.width,this.props.height)/2-10

    var a = arc()
            .innerRadius(radius-40)
            .outerRadius(radius)
            .cornerRadius(5);

    var p = pie()
            .padAngle(.02)
    
    var tot = this.state.data.reduce( (acc,val) => acc+val ) 

    var c = this.state.data.map( (d,i) => scaleLinear()
                                            .domain([0,tot])
                                            .range(['red','green'])(d))
    c[0] = '#DDD'

    var segments = p(this.state.data).map( (d,i) => <path style={{fill: c[i]}} d={a(d)}/> )

    return <div className={classes.root}>
            <svg width={this.props.width} height={this.props.height}>
            <g transform={`translate(${this.props.width/2},${this.props.height/2})`}>
              {segments}
              <g transform="translate(0,5)">
              <text style={{fontSize: '40px', fill: c[1]}} text-anchor="middle">{this.state.data[1]}/{tot}</text>
              </g>
              <g transform="translate(0,30)">
              <text style={{fontSize: '20px', fill: c[1]}} text-anchor="middle">available</text>
              </g>
            </g>
            </svg>
            <Websocket url={this.props.endpoint}
             onMessage={this.handleData.bind(this)}/>
          </div>
  }

}

export default withStyles(style)(MacPlot)
