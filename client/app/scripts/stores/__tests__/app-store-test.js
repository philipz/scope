

describe('AppStore', function() {
  const ActionTypes = require('../../constants/action-types');
  let AppStore;
  let registeredCallback;

  // fixtures

  const NODE_SET = {
    n1: {
      id: 'n1',
      rank: undefined,
      adjacency: ['n1', 'n2'],
      pseudo: undefined,
      label_major: undefined,
      label_minor: undefined
    },
    n2: {
      id: 'n2',
      rank: undefined,
      adjacency: undefined,
      pseudo: undefined,
      label_major: undefined,
      label_minor: undefined
    }
  };

  // actions

  const ChangeTopologyOptionAction = {
    type: ActionTypes.CHANGE_TOPOLOGY_OPTION,
    option: 'option1',
    value: 'off'
  };

  const ChangeTopologyOptionAction2 = {
    type: ActionTypes.CHANGE_TOPOLOGY_OPTION,
    option: 'option1',
    value: 'on'
  };

  const ClickNodeAction = {
    type: ActionTypes.CLICK_NODE,
    nodeId: 'n1'
  };

  const ClickSubTopologyAction = {
    type: ActionTypes.CLICK_TOPOLOGY,
    topologyId: 'topo1-grouped'
  };

  const ClickTopologyAction = {
    type: ActionTypes.CLICK_TOPOLOGY,
    topologyId: 'topo1'
  };

  const ClickGroupingAction = {
    type: ActionTypes.CLICK_GROUPING,
    grouping: 'grouped'
  };

  const CloseWebsocketAction = {
    type: ActionTypes.CLOSE_WEBSOCKET
  };

  const HitEscAction = {
    type: ActionTypes.HIT_ESC_KEY
  };

  const OpenWebsocketAction = {
    type: ActionTypes.OPEN_WEBSOCKET
  };

  const ReceiveNodesDeltaAction = {
    type: ActionTypes.RECEIVE_NODES_DELTA,
    delta: {
      add: [{
        id: 'n1',
        adjacency: ['n1', 'n2']
      }, {
        id: 'n2'
      }]
    }
  };

  const ReceiveNodesDeltaUpdateAction = {
    type: ActionTypes.RECEIVE_NODES_DELTA,
    delta: {
      update: [{
        id: 'n1',
        adjacency: ['n1']
      }],
      remove: ['n2']
    }
  };

  const ReceiveTopologiesAction = {
    type: ActionTypes.RECEIVE_TOPOLOGIES,
    topologies: [{
      url: '/topo1',
      name: 'Topo1',
      options: {
        option1: [
          {value: 'on', default: true},
          {value: 'off'}
        ]
      },
      sub_topologies: [{
        url: '/topo1-grouped',
        name: 'topo 1 grouped'
      }]
    }]
  };

  const RouteAction = {
    type: ActionTypes.ROUTE_TOPOLOGY,
    state: {}
  };

  beforeEach(function() {
    // clear AppStore singleton
    delete require.cache[require.resolve('../app-store')];
    AppStore = require('../app-store');
    registeredCallback = AppStore.registeredCallback;
  });

  // topology tests

  it('init with no topologies', function() {
    const topos = AppStore.getTopologies();
    expect(topos.length).toBe(0);
    expect(AppStore.getCurrentTopology()).toBeUndefined();
  });

  it('get current topology', function() {
    registeredCallback(ClickTopologyAction);
    registeredCallback(ReceiveTopologiesAction);

    expect(AppStore.getTopologies().length).toBe(1);
    expect(AppStore.getCurrentTopology().name).toBe('Topo1');
    expect(AppStore.getCurrentTopologyUrl()).toBe('/topo1');
    expect(AppStore.getCurrentTopologyOptions().option1).toBeDefined();
  });

  it('get sub-topology', function() {
    registeredCallback(ReceiveTopologiesAction);
    registeredCallback(ClickSubTopologyAction);

    expect(AppStore.getTopologies().length).toBe(1);
    expect(AppStore.getCurrentTopology().name).toBe('topo 1 grouped');
    expect(AppStore.getCurrentTopologyUrl()).toBe('/topo1-grouped');
    expect(AppStore.getCurrentTopologyOptions()).toBeUndefined();
  });

  // topology options

  it('changes topology option', function() {
    registeredCallback(ReceiveTopologiesAction);
    // default options just dont show
    expect(AppStore.getActiveTopologyOptions()).toEqual({});
    expect(AppStore.getAppState().topologyOptions.option1).toBeUndefined();

    registeredCallback(ChangeTopologyOptionAction);
    expect(AppStore.getActiveTopologyOptions()).toEqual({option1: 'off'});
    expect(AppStore.getAppState().topologyOptions.option1).toBe('off');

    registeredCallback(ChangeTopologyOptionAction2);
    expect(AppStore.getActiveTopologyOptions()).toEqual({option1: 'on'});
    expect(AppStore.getAppState().topologyOptions.option1).toBe('on');
  });

  // nodes delta

  it('replaces adjacency on update', function() {
    registeredCallback(ReceiveNodesDeltaAction);
    expect(AppStore.getNodes().toJS().n1.adjacency).toEqual(['n1', 'n2']);
    registeredCallback(ReceiveNodesDeltaUpdateAction);
    expect(AppStore.getNodes().toJS().n1.adjacency).toEqual(['n1']);
  });

  // browsing

  it('shows nodes that were received', function() {
    registeredCallback(ReceiveNodesDeltaAction);
    expect(AppStore.getNodes().toJS()).toEqual(NODE_SET);
  });

  it('gets selected node after click', function() {
    registeredCallback(ReceiveNodesDeltaAction);

    registeredCallback(ClickNodeAction);
    expect(AppStore.getSelectedNodeId()).toBe('n1');
    expect(AppStore.getNodes().toJS()).toEqual(NODE_SET);

    registeredCallback(HitEscAction)
    expect(AppStore.getSelectedNodeId()).toBe(null);
    expect(AppStore.getNodes().toJS()).toEqual(NODE_SET);
  });

  it('keeps showing nodes on navigating back after node click', function() {
    registeredCallback(ReceiveTopologiesAction);
    registeredCallback(ClickTopologyAction);
    registeredCallback(ReceiveNodesDeltaAction);

    expect(AppStore.getAppState().selectedNodeId).toEqual(null);

    registeredCallback(ClickNodeAction);
    expect(AppStore.getAppState().selectedNodeId).toEqual('n1');

    // go back in browsing
    RouteAction.state = {"topologyId":"topo1","selectedNodeId": null};
    registeredCallback(RouteAction);
    expect(AppStore.getSelectedNodeId()).toBe(null);
    expect(AppStore.getNodes().toJS()).toEqual(NODE_SET);
  });

  it('closes details when changing topologies', function() {
    registeredCallback(ReceiveTopologiesAction);
    registeredCallback(ClickTopologyAction);
    registeredCallback(ReceiveNodesDeltaAction);

    expect(AppStore.getAppState().selectedNodeId).toEqual(null);
    expect(AppStore.getAppState().topologyId).toEqual('topo1');

    registeredCallback(ClickNodeAction);
    expect(AppStore.getAppState().selectedNodeId).toEqual('n1');
    expect(AppStore.getAppState().topologyId).toEqual('topo1');

    registeredCallback(ClickSubTopologyAction);
    expect(AppStore.getAppState().selectedNodeId).toEqual(null);
    expect(AppStore.getAppState().topologyId).toEqual('topo1-grouped');
  });

  // connection errors

  it('resets topology on websocket reconnect', function() {
    registeredCallback(ReceiveNodesDeltaAction);
    expect(AppStore.getNodes().toJS()).toEqual(NODE_SET);

    registeredCallback(CloseWebsocketAction);
    expect(AppStore.isWebsocketClosed()).toBeTruthy();
    // keep showing old nodes
    expect(AppStore.getNodes().toJS()).toEqual(NODE_SET);

    registeredCallback(OpenWebsocketAction);
    expect(AppStore.isWebsocketClosed()).toBeFalsy();
    // opened socket clears nodes
    expect(AppStore.getNodes().toJS()).toEqual({});
  });


});