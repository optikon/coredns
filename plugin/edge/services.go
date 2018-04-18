package edge

// Starts the process of reading Kubernetes services every read interval.
func (oe *OptikonEdge) startReadingServices() {
	oe.svcReadProbe.Start(oe.svcReadInterval)
	oe.svcReadProbe.Do(readServices)
}

// Stops reading Kubernetes services into local state.
func (oe *OptikonEdge) stopReadingServices() {
	oe.svcReadProbe.Stop()
}

// Reads Kubernetes services using the KubeAPI.
func readServices() error {

}

func (p *Proxy) startPushingServices(services *ConcurrentStringSet) {

}

func (p *Proxy) stopPushingServices() {

}
