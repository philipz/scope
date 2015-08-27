const _ = require('lodash');
const d3 = require('d3');
const debug = require('debug')('scope:nodes-chart');
const React = require('react');
const md5 = require('md5');

const Edge = require('./edge');
const Naming = require('../constants/naming');
const NodesLayout = require('./nodes-layout');
const Node = require('./node');

const MARGINS = {
  top: 100,
  left: 40,
  right: 40,
  bottom: 20
};

const NodesChart = React.createClass({

  getInitialState: function() {
    return {
      nodes: [],
      edges: [],
      nodeScale: 1,
      translate: [MARGINS.left, MARGINS.top],
      scale: 1,
      initialLayout: true,
      hasZoomed: false
    };
  },

  componentDidMount: function() {
    this.zoom = d3.behavior.zoom()
      .scaleExtent([0.1, 2])
      .on('zoom', this.zoomed);

    d3.select('.nodes-chart')
      .call(this.zoom);

    this.updateGraphState(this.props);
  },

  componentWillReceiveProps: function(nextProps) {
    // debug('fingerprint',
    //   _.size(nextProps.nodes), md5(this.getTopologyFingerprint(nextProps.nodes)),
    //   _.size(this.props.nodes), md5(this.getTopologyFingerprint(this.props.nodes)));
    if (nextProps.nodes !== this.props.nodes) {
      this.setState({
        hasZoomed: false,
        initialLayout: true
      });
      this.updateGraphState(nextProps);
    }
  },

  componentWillUnmount: function() {
    // undoing .call(zoom)

    d3.select('.nodes-chart')
      .on('mousedown.zoom', null)
      .on('onwheel', null)
      .on('onmousewheel', null)
      .on('dblclick.zoom', null)
      .on('touchstart.zoom', null);
  },

  getTopologyFingerprint: function(topology) {
    const fingerprint = [];

    _.each(topology, function(node) {
      fingerprint.push(node.id);
      if (node.adjacency) {
        fingerprint.push(node.adjacency.join(','));
      }
    });
    return fingerprint.join(';');
  },

  renderGraphNodes: function(nodes, scale) {
    return _.map(nodes, function(node) {
      const highlighted = _.includes(this.props.highlightedNodeIds, node.id);
      return (
        <Node
          highlighted={highlighted}
          onClick={this.props.onNodeClick}
          key={node.id}
          id={node.id}
          label={node.label}
          pseudo={node.pseudo}
          subLabel={node.subLabel}
          scale={scale}
          dx={node.x}
          dy={node.y}
        />
      );
    }, this);
  },

  renderGraphEdges: function() {
    const edges = this.state.edges;

    return _.map(edges, function(edge) {
      const highlighted = _.includes(this.props.highlightedEdgeIds, edge.id);
      const points = [{
        x: edge.source.x,
        y: edge.source.y
      }, {
        x: edge.target.x,
        y: edge.target.y
      }];
      return (
        <Edge key={edge.id} id={edge.id} points={points} highlighted={highlighted} />
      );
    }, this);
  },

  render: function() {
    const nodeElements = this.renderGraphNodes(this.state.nodes, this.state.nodeScale);
    const edgeElements = this.renderGraphEdges();
    const transform = 'translate(' + this.state.translate + ')' +
      ' scale(' + this.state.scale + ')';

    return (
      <svg width="100%" height="100%" className="nodes-chart">
        <g className="canvas" transform={transform}>
          <g className="edges">
            {edgeElements}
          </g>
          <g className="nodes">
            {nodeElements}
          </g>
        </g>
      </svg>
    );
  },

  initNodes: function(topology, prevNodes) {
    const centerX = this.props.width / 2;
    const centerY = this.props.height / 2;
    const nodes = {};

    topology.forEach(function(node, id) {
      nodes[id] = {};

      // use cached positions if available
      _.defaults(nodes[id], prevNodes[id], {
        x: centerX,
        y: centerY
      });

      // copy relevant fields to state nodes
      _.assign(nodes[id], {
        id: id,
        label: node.get('label_major'),
        pseudo: node.get('pseudo'),
        subLabel: node.get('label_minor'),
        rank: node.get('rank')
      });
    });

    return nodes;
  },

  initEdges: function(topology, nodes) {
    const edges = {};

    topology.forEach(function(node, nodeId) {
      const adjacency = node.get('adjacency');
      if (adjacency) {
        adjacency.forEach(function(adjacent) {
          const edge = [nodeId, adjacent];
          const edgeId = edge.join(Naming.EDGE_ID_SEPARATOR);

          if (!edges[edgeId]) {
            const source = nodes[edge[0]];
            const target = nodes[edge[1]];

            if (!source || !target) {
              debug('Missing edge node', edge[0], source, edge[1], target);
            }

            edges[edgeId] = {
              id: edgeId,
              value: 1,
              source: target,
              target: source
            };
          }
        });
      }
    });

    return edges;
  },

  updateGraphState: function(props) {
    const n = props.nodes.size;

    if (n === 0) {
      return;
    }

    const nodes = this.initNodes(props.nodes, this.state.nodes);
    const edges = this.initEdges(props.nodes, nodes);
    const width = props.width - MARGINS.left - MARGINS.right;
    const height = props.height - MARGINS.top - MARGINS.bottom;
    const expanse = Math.min(props.height, props.width);
    const nodeSize = expanse / 2;
    const nodeScale = d3.scale.linear().range([0, nodeSize / Math.pow(n, 0.7)]);

    debug('fingerprint', md5(JSON.stringify(_.keys(nodes)) + JSON.stringify(_.keys(edges))));

    let graph = NodesLayout.doLayout(nodes, edges, width, height, nodeScale);
    if (this.state.initialLayout && graph.width > 0) {
      debug('running layout twice to reduce jitter on initial layout');
      graph = NodesLayout.doLayout(nodes, edges, width, height, nodeScale);
    }

    // adjust layout based on viewport
    const empty = graph.width === 0;
    const xFactor = width / graph.width;
    const yFactor = height / graph.height;
    const xOffset = graph.left;
    const yOffset = graph.top;
    // only adjust zooming in
    const zoomFactor = Math.min(1, Math.min(xFactor, yFactor));
    let zoomScale = this.state.scale;
    let translate = this.state.translate;

    if (this.zoom && !this.state.hasZoomed) {
      let adjusted = false;

      if (zoomFactor > 0 && zoomFactor !== zoomScale) {
        zoomScale = zoomFactor;
        // saving in d3's behavior cache
        this.zoom.scale(zoomFactor);
        adjusted = true;
      }

      if (xOffset < 0) {
        translate[0] = xOffset * -1 * zoomScale + MARGINS.left;
        adjusted = true;
      }

      if (yOffset < 0) {
        translate[1] = yOffset * -1 * zoomScale + MARGINS.top;
        adjusted = true;
      }

      // saving in d3's behavior cache
      this.zoom.translate(translate);

      if (adjusted) {
        debug('adjust graph', graph, translate, zoomScale);
      }
    }

    this.setState({
      nodes: nodes,
      edges: edges,
      nodeScale: nodeScale,
      scale: zoomScale,
      translate: translate,
      initialLayout: empty
    });
  },

  zoomed: function() {
    this.setState({
      hasZoomed: true,
      translate: d3.event.translate,
      scale: d3.event.scale
    });
  }

});

module.exports = NodesChart;
