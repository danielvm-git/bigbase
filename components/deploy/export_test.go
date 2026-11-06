package deploy

// PickPort exposes pickPort for whitebox unit testing in deploy_test package.
func PickPort(base int) (int, error) {
	return pickPort(base)
}

// DrainForTest exposes drainDeployment for whitebox unit testing in the
// deploy_test package (unexported lifecycle path).
func (d *Deploy) DrainForTest(id string) {
	d.drainDeployment(id)
}
