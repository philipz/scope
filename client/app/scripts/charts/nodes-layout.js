const _ = require('lodash');
const d3 = require('d3');
const webcola = require('webcola');
const debug = require('debug')('scope:nodes-layout');

const MAX_NODES = 100;

const getConstraints = function(nodes) {
  const constraints = [];
  const theInternetId = 'theinternet';
  const internetNode = nodes[theInternetId];

  // all nodes have to be below the internet node
  _.each(nodes, function(node, id) {
    if (id !== theInternetId) {
      constraints.push({
        axis: 'y',
        left: internetNode.index,
        right: node.index,
        gap: 200
      });
    }
  });

  return constraints;
};

const doLayout = function(nodes, edges, width, height, scale) {
  if (_.size(nodes) > MAX_NODES) {
    debug('Too many nodes to lay out.');
    return {height: 0, width: 0};
  }

  if (_.size(nodes) === 0) {
    return {height: 0, width: 0};
  }

  const cola = new webcola.Layout()
    .avoidOverlaps(true)
    .size([width, height]);

  const nodeList = _.sortBy(nodes, function(node) {
    return node.id;
  });
  const edgeList = _.values(edges, function(edge) {
    return edge.id;
  });

  nodeList.forEach(function(v, i) {
    v.height = scale(2.25);
    v.width = scale(2.25);
    v.index = i;
  });

  const constraints = getConstraints(nodes);

  debug('graph layout constraints', constraints);

  cola
    .nodes(nodeList)
    .links(edgeList)
    .constraints(constraints)
    .flowLayout('y', 25)
    .start(5, 10, 20);

  debug('graph layout done');

  const extentX = d3.extent(nodeList, function(n) { return n.x; });
  const extentY = d3.extent(nodeList, function(n) { return n.y; });

  // return object with the width and height of layout

  return {
    left: extentX[0],
    height: extentY[1] - extentY[0],
    top: extentY[0],
    width: extentX[1] - extentX[0]
  };
};

module.exports = {
  doLayout: doLayout
};
