package edge

import (
	"errors"
	"fmt"
)

var (
	errInvalidDomain         = errors.New("invalid domain for forward")
	errNoHealthy             = errors.New("no healthy proxies")
	errNoEdge                = fmt.Errorf("no %s defined", pluginName)
	errTableParseFailure     = errors.New("unable to parse Table returned from upstream")
	errFindingClosestCluster = errors.New("unable to compute closest edge cluster")
	errInvalidIP             = errors.New("invalid IP address")
)
